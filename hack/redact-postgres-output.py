#!/usr/bin/env python3
"""Redact captured launcher output; connection secrets arrive only via env."""

import json
import os
import re
import sys
from urllib.parse import parse_qsl, unquote, urlsplit


def passwords(dsn):
    if dsn.startswith(("postgres://", "postgresql://")):
        url = urlsplit(dsn)
        result = [unquote(url.password)] if url.password is not None else []
        result.extend(value for key, value in parse_qsl(url.query) if key == "password")
        return result

    # Match lib/pq's keyword scanner: whitespace around '=', optional quoted
    # values, and backslash escaping in both quoted and unquoted values.
    result = []
    pos = 0
    while pos < len(dsn):
        match = re.match(r"\s*([^\s=]+)\s*=\s*", dsn[pos:])
        if match is None:
            if dsn[pos:].strip():
                raise ValueError("invalid connection options")
            break
        key = match[1]
        pos += match.end()
        quoted = pos < len(dsn) and dsn[pos] == "'"
        if quoted:
            pos += 1
        value = []
        closed = not quoted
        while pos < len(dsn):
            char = dsn[pos]
            pos += 1
            if char == "\\":
                if pos == len(dsn):
                    raise ValueError("incomplete escape")
                value.append(dsn[pos])
                pos += 1
            elif quoted and char == "'":
                closed = True
                break
            elif not quoted and char.isspace():
                break
            else:
                value.append(char)
        if not closed:
            raise ValueError("unterminated value")
        if key == "password":
            result.append("".join(value))
    return result


def main():
    dsn = os.environ.get("TRAGGO_TEST_POSTGRES_DSN", "")
    try:
        secrets = [dsn] + passwords(dsn)
    except ValueError:
        # Never replay captured diagnostics if we cannot identify credentials.
        print("error: output suppressed: cannot safely parse PostgreSQL connection settings", file=sys.stderr)
        return 1

    replacements = {}
    for secret in secrets:
        if secret:
            marker = "<redacted DSN>" if secret == dsn else "<redacted>"
            replacements[secret] = marker
            # go test -json escapes quotes, backslashes and control characters.
            for ascii_only in (False, True):
                escaped = json.dumps(secret, ensure_ascii=ascii_only)[1:-1]
                replacements[escaped] = marker
                replacements[escaped.replace("<", r"\u003c").replace(">", r"\u003e").replace("&", r"\u0026")] = marker
    # One pass avoids a later replacement matching text inside our markers.
    if replacements:
        pattern = re.compile("|".join(re.escape(value) for value in sorted(replacements, key=len, reverse=True)))
        output = pattern.sub(lambda match: replacements[match[0]], sys.stdin.read())
    else:
        output = sys.stdin.read()
    sys.stdout.write(output)
    return 0


if __name__ == "__main__":
    sys.exit(main())
