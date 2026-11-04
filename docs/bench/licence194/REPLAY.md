verdict: INCONCLUSIVE W=238 R=0 U=137

ADR-194 T1: the narrow licence replayed on the Stage 8 pilot transcripts, against the criterion
pre-registered in docs/adr/BACKLOG.md (From ADR-194). Script: scripts/licence194_replay.py;
corpus: 73 transcripts, manifest sha256 364d95c14fe8722c65268392de9cdc95ac307a7247ad863316bd84de9852fb20.

| cell | W | R | U |
|------|---|---|---|
| claude-opus-5-5 B | 91 | 0 | 60 |
| claude-opus-5-5 S | 90 | 0 | 53 |
| claude-sonnet-5-5 B | 23 | 0 | 13 |
| claude-sonnet-5-5 S | 34 | 0 | 11 |

| attempt | kind | file | line | op | reason |
|---|---|---|---|---|---|
| astropy__astropy-14539.B.claude-opus-5-5.1 | U | astropy/io/fits/tests/test_diff.py | 458 | insert-after | the file is UNCERTAIN |
| django__django-11451.S.claude-opus-5-5.1 | U | tests/auth_tests/test_auth_backends.py | 226 | replace | the file is UNCERTAIN |
| django__django-11532.B.claude-opus-5-5.1 | U | django/core/mail/message.py | 19 | replace | the file is UNCERTAIN |
| django__django-11532.B.claude-opus-5-5.1 | U | django/core/mail/message.py | 105 | replace | the file is UNCERTAIN |
| django__django-11532.B.claude-opus-5-5.1 | U | django/core/mail/message.py | 260 | replace | the file is UNCERTAIN |
| django__django-11532.B.claude-opus-5-5.1 | U | django/core/mail/message.py | 257 | replace | the file is UNCERTAIN |
| django__django-11532.B.claude-opus-5-5.1 | U | django/core/mail/message.py | 256 | delete | the file is UNCERTAIN |
| django__django-11532.B.claude-opus-5-5.1 | U | tests/mail/tests.py | None | terse | a --terse receipt names no hunk address |
| django__django-11532.B.claude-opus-5-5.1 | U | tests/mail/tests.py | None | terse | a --terse receipt names no hunk address |
| django__django-11532.B.claude-opus-5-5.1 | U | tests/mail/tests.py | None | terse | a --terse receipt names no hunk address |
| django__django-11532.S.claude-opus-5-5.1 | U | django/utils/encoding.py | 221 | replace | the file is UNCERTAIN |
| django__django-11532.S.claude-opus-5-5.1 | U | django/core/mail/message.py | 19 | replace | the file is UNCERTAIN |
| django__django-11532.S.claude-opus-5-5.1 | U | django/core/mail/message.py | 108 | replace | the file is UNCERTAIN |
| django__django-11532.S.claude-opus-5-5.1 | U | django/core/mail/message.py | 260 | replace | the file is UNCERTAIN |
| django__django-11532.S.claude-opus-5-5.1 | U | tests/mail/tests.py | 15 | replace | the file is UNCERTAIN |
| django__django-11532.S.claude-opus-5-5.1 | U | tests/mail/tests.py | 24 | replace | the file is UNCERTAIN |
| django__django-11532.S.claude-opus-5-5.1 | U | tests/mail/tests.py | 301 | replace | the file is UNCERTAIN |
| django__django-11555.B.claude-opus-5-5.1 | U | django/db/models/sql/compiler.py | 724 | replace | the file is UNCERTAIN |
| django__django-11555.B.claude-opus-5-5.1 | U | tests/ordering/models.py | 56 | insert-before | the file is UNCERTAIN |
| django__django-11555.B.claude-opus-5-5.1 | U | tests/ordering/tests.py | 12 | replace | the file is UNCERTAIN |
| django__django-11555.B.claude-opus-5-5.1 | U | tests/ordering/tests.py | 464 | insert-after | the file is UNCERTAIN |
| django__django-11555.B.claude-opus-5-5.1 | U | django/db/models/sql/compiler.py | 725 | delete | the file is UNCERTAIN |
| django__django-11555.B.claude-opus-5-5.1 | U | django/db/models/sql/compiler.py | 724 | insert-after | the file is UNCERTAIN |
| django__django-14011.B.claude-opus-5-5.1 | U | django/core/servers/basehttp.py | None | terse | a --terse receipt names no hunk address |
| django__django-14011.B.claude-opus-5-5.1 | U | django/core/servers/basehttp.py | None | terse | a --terse receipt names no hunk address |
| django__django-14011.B.claude-opus-5-5.1 | U | django/test/testcases.py | None | terse | a --terse receipt names no hunk address |
| django__django-14011.B.claude-opus-5-5.1 | U | django/test/testcases.py | None | terse | a --terse receipt names no hunk address |
| django__django-14011.B.claude-opus-5-5.1 | U | tests/servers/tests.py | None | terse | a --terse receipt names no hunk address |
| django__django-14011.B.claude-opus-5-5.1 | U | tests/servers/tests.py | None | terse | a --terse receipt names no hunk address |
| django__django-14011.S.claude-opus-5-5.1 | U | django/core/servers/basehttp.py | None | terse | a --terse receipt names no hunk address |
| django__django-14011.S.claude-opus-5-5.1 | U | django/core/servers/basehttp.py | None | terse | a --terse receipt names no hunk address |
| django__django-14011.S.claude-opus-5-5.1 | U | django/test/testcases.py | None | terse | a --terse receipt names no hunk address |
| django__django-14011.S.claude-opus-5-5.1 | U | django/test/testcases.py | None | terse | a --terse receipt names no hunk address |
| django__django-14011.S.claude-opus-5-5.1 | U | tests/servers/tests.py | None | terse | a --terse receipt names no hunk address |
| django__django-14011.S.claude-opus-5-5.1 | U | tests/servers/tests.py | None | terse | a --terse receipt names no hunk address |
| django__django-14011.S.claude-opus-5-5.1 | U | tests/servers/test_liveserverthread.py | None | terse | a --terse receipt names no hunk address |
| django__django-14011.S.claude-opus-5-5.1 | U | tests/servers/test_liveserverthread.py | None | terse | a --terse receipt names no hunk address |
| django__django-14404.B.claude-opus-5-5.1 | U | admin_views/tests.py | None | terse | a --terse receipt names no hunk address |
| django__django-16116.B.claude-opus-5-5.1 | U | django/core/management/commands/makemigrations.py | 113 | replace | the file is UNCERTAIN |
| django__django-16116.B.claude-opus-5-5.1 | U | docs/ref/django-admin.txt | 825 | replace | the file is UNCERTAIN |
| django__django-16116.B.claude-opus-5-5.1 | U | tests/migrations/test_commands.py | 2394 | replace | the file is UNCERTAIN |
| django__django-16116.S.claude-opus-5-5.1 | U | django/core/management/commands/makemigrations.py | 113 | replace | the file is UNCERTAIN |
| django__django-16116.S.claude-opus-5-5.1 | U | django/core/management/commands/makemigrations.py | 73 | replace | the file is UNCERTAIN |
| django__django-16642.S.claude-opus-5-5.1 | U | tests/responses/test_fileresponse.py | 257 | replace | the file is UNCERTAIN |
| matplotlib__matplotlib-25775.B.claude-opus-5-5.1 | U | lib/matplotlib/text.py | None | terse | a --terse receipt names no hunk address |
| matplotlib__matplotlib-25775.B.claude-opus-5-5.1 | U | lib/matplotlib/text.py | None | terse | a --terse receipt names no hunk address |
| matplotlib__matplotlib-25775.B.claude-opus-5-5.1 | U | lib/matplotlib/text.py | None | terse | a --terse receipt names no hunk address |
| matplotlib__matplotlib-25775.B.claude-opus-5-5.1 | U | lib/matplotlib/text.py | None | terse | a --terse receipt names no hunk address |
| matplotlib__matplotlib-25775.B.claude-opus-5-5.1 | U | lib/matplotlib/text.py | None | terse | a --terse receipt names no hunk address |
| matplotlib__matplotlib-25775.B.claude-opus-5-5.1 | U | lib/matplotlib/text.py | None | terse | a --terse receipt names no hunk address |
| matplotlib__matplotlib-25775.B.claude-opus-5-5.1 | U | lib/matplotlib/text.py | None | terse | a --terse receipt names no hunk address |
| matplotlib__matplotlib-25775.B.claude-opus-5-5.1 | U | lib/matplotlib/text.pyi | None | terse | a --terse receipt names no hunk address |
| matplotlib__matplotlib-25775.B.claude-opus-5-5.1 | U | lib/matplotlib/text.pyi | None | terse | a --terse receipt names no hunk address |
| matplotlib__matplotlib-25775.B.claude-opus-5-5.1 | U | lib/matplotlib/backends/backend_cairo.py | None | terse | a --terse receipt names no hunk address |
| matplotlib__matplotlib-25775.B.claude-opus-5-5.1 | U | lib/matplotlib/backends/backend_agg.py | None | terse | a --terse receipt names no hunk address |
| matplotlib__matplotlib-25775.B.claude-opus-5-5.1 | U | lib/matplotlib/mathtext.py | None | terse | a --terse receipt names no hunk address |
| matplotlib__matplotlib-25775.B.claude-opus-5-5.1 | U | lib/matplotlib/mathtext.py | None | terse | a --terse receipt names no hunk address |
| matplotlib__matplotlib-25775.B.claude-opus-5-5.1 | U | lib/matplotlib/mathtext.py | None | terse | a --terse receipt names no hunk address |
| matplotlib__matplotlib-25775.B.claude-opus-5-5.1 | U | lib/matplotlib/mathtext.py | None | terse | a --terse receipt names no hunk address |
| matplotlib__matplotlib-25775.B.claude-opus-5-5.1 | U | lib/matplotlib/mathtext.py | None | terse | a --terse receipt names no hunk address |
| matplotlib__matplotlib-25775.B.claude-opus-5-5.1 | U | lib/matplotlib/mathtext.py | None | terse | a --terse receipt names no hunk address |
| matplotlib__matplotlib-25775.B.claude-opus-5-5.1 | U | lib/matplotlib/mathtext.pyi | None | terse | a --terse receipt names no hunk address |
| matplotlib__matplotlib-25775.B.claude-opus-5-5.1 | U | lib/matplotlib/_mathtext.py | None | terse | a --terse receipt names no hunk address |
| matplotlib__matplotlib-25775.B.claude-opus-5-5.1 | U | lib/matplotlib/_mathtext.py | None | terse | a --terse receipt names no hunk address |
| matplotlib__matplotlib-25775.B.claude-opus-5-5.1 | U | lib/matplotlib/backends/backend_agg.py | None | terse | a --terse receipt names no hunk address |
| matplotlib__matplotlib-25775.B.claude-opus-5-5.1 | U | lib/matplotlib/text.py | None | terse | a --terse receipt names no hunk address |
| matplotlib__matplotlib-25775.B.claude-opus-5-5.1 | U | lib/matplotlib/text.py | None | terse | a --terse receipt names no hunk address |
| matplotlib__matplotlib-25775.B.claude-opus-5-5.1 | U | lib/matplotlib/tests/test_text.py | None | terse | a --terse receipt names no hunk address |
| matplotlib__matplotlib-25775.B.claude-opus-5-5.1 | U | lib/matplotlib/tests/test_text.py | None | terse | a --terse receipt names no hunk address |
| matplotlib__matplotlib-25775.B.claude-opus-5-5.1 | U | doc/users/next_whats_new/text_antialiasing.rst | None | terse | a --terse receipt names no hunk address |
| matplotlib__matplotlib-25775.S.claude-opus-5-5.1 | U | text.py | None | terse | a --terse receipt names no hunk address |
| matplotlib__matplotlib-25775.S.claude-opus-5-5.1 | U | text.py | None | terse | a --terse receipt names no hunk address |
| matplotlib__matplotlib-25775.S.claude-opus-5-5.1 | U | text.py | None | terse | a --terse receipt names no hunk address |
| matplotlib__matplotlib-25775.S.claude-opus-5-5.1 | U | text.py | None | terse | a --terse receipt names no hunk address |
| matplotlib__matplotlib-25775.S.claude-opus-5-5.1 | U | text.py | None | terse | a --terse receipt names no hunk address |
| matplotlib__matplotlib-25775.S.claude-opus-5-5.1 | U | text.py | None | terse | a --terse receipt names no hunk address |
| matplotlib__matplotlib-25775.S.claude-opus-5-5.1 | U | text.py | None | terse | a --terse receipt names no hunk address |
| matplotlib__matplotlib-25775.S.claude-opus-5-5.1 | U | text.pyi | None | terse | a --terse receipt names no hunk address |
| matplotlib__matplotlib-25775.S.claude-opus-5-5.1 | U | text.pyi | None | terse | a --terse receipt names no hunk address |
| matplotlib__matplotlib-25775.S.claude-opus-5-5.1 | U | backends/backend_agg.py | None | terse | a --terse receipt names no hunk address |
| matplotlib__matplotlib-25775.S.claude-opus-5-5.1 | U | backends/backend_agg.py | None | terse | a --terse receipt names no hunk address |
| matplotlib__matplotlib-25775.S.claude-opus-5-5.1 | U | backends/backend_cairo.py | None | terse | a --terse receipt names no hunk address |
| matplotlib__matplotlib-25775.S.claude-opus-5-5.1 | U | backends/backend_cairo.py | None | terse | a --terse receipt names no hunk address |
| matplotlib__matplotlib-25775.S.claude-opus-5-5.1 | U | mathtext.py | None | terse | a --terse receipt names no hunk address |
| matplotlib__matplotlib-25775.S.claude-opus-5-5.1 | U | mathtext.py | None | terse | a --terse receipt names no hunk address |
| matplotlib__matplotlib-25775.S.claude-opus-5-5.1 | U | mathtext.py | None | terse | a --terse receipt names no hunk address |
| matplotlib__matplotlib-25775.S.claude-opus-5-5.1 | U | mathtext.py | None | terse | a --terse receipt names no hunk address |
| matplotlib__matplotlib-25775.S.claude-opus-5-5.1 | U | mathtext.pyi | None | terse | a --terse receipt names no hunk address |
| matplotlib__matplotlib-25775.S.claude-opus-5-5.1 | U | _mathtext.py | None | terse | a --terse receipt names no hunk address |
| matplotlib__matplotlib-25775.S.claude-opus-5-5.1 | U | _mathtext.py | None | terse | a --terse receipt names no hunk address |
| matplotlib__matplotlib-25775.S.claude-opus-5-5.1 | U | mathtext.py | None | terse | a --terse receipt names no hunk address |
| matplotlib__matplotlib-25775.S.claude-opus-5-5.1 | U | mathtext.py | None | terse | a --terse receipt names no hunk address |
| matplotlib__matplotlib-25775.S.claude-opus-5-5.1 | U | text.py | None | terse | a --terse receipt names no hunk address |
| matplotlib__matplotlib-25775.S.claude-opus-5-5.1 | U | lib/matplotlib/tests/test_text.py | None | terse | a --terse receipt names no hunk address |
| matplotlib__matplotlib-25775.S.claude-opus-5-5.1 | U | lib/matplotlib/tests/test_text.py | None | terse | a --terse receipt names no hunk address |
| pytest-dev__pytest-5631.B.claude-opus-5-5.1 | U | src/_pytest/compat.py | 70 | replace | the file is UNCERTAIN |
| pytest-dev__pytest-5631.B.claude-opus-5-5.1 | U | testing/python/integration.py | 157 | insert-after | the file is UNCERTAIN |
| pytest-dev__pytest-5631.B.claude-opus-5-5.1 | U | testing/python/integration.py | 158 | replace | the file is UNCERTAIN |
| pytest-dev__pytest-5631.S.claude-opus-5-5.1 | U | src/_pytest/compat.py | 71 | replace | the file is UNCERTAIN |
| pytest-dev__pytest-5631.S.claude-opus-5-5.1 | U | testing/python/integration.py | 155 | replace | the file is UNCERTAIN |
| scikit-learn__scikit-learn-12585.B.claude-opus-5-5.1 | U | sklearn/tests/test_base.py | 168 | insert-after | the file is UNCERTAIN |
| scikit-learn__scikit-learn-12585.S.claude-opus-5-5.1 | U | sklearn/tests/test_base.py | 146 | replace | the file is UNCERTAIN |
| scikit-learn__scikit-learn-25747.B.claude-opus-5-5.1 | U | sklearn/utils/tests/test_set_output.py | 27 | replace | the file is UNCERTAIN |
| scikit-learn__scikit-learn-25747.B.claude-opus-5-5.1 | U | sklearn/tests/test_pipeline.py | 1621 | insert-after | the file is UNCERTAIN |
| sphinx-doc__sphinx-8265.B.claude-opus-5-5.1 | U | sphinx/pycode/ast.py | None | terse | a --terse receipt names no hunk address |
| sphinx-doc__sphinx-8265.B.claude-opus-5-5.1 | U | sphinx/pycode/ast.py | None | terse | a --terse receipt names no hunk address |
| sphinx-doc__sphinx-8265.B.claude-opus-5-5.1 | U | tests/test_pycode_ast.py | None | terse | a --terse receipt names no hunk address |
| sphinx-doc__sphinx-8265.S.claude-opus-5-5.1 | U | sphinx/pycode/ast.py | 168 | replace | the file is UNCERTAIN |
| sphinx-doc__sphinx-8265.S.claude-opus-5-5.1 | U | sphinx/pycode/ast.py | 175 | replace | the file is UNCERTAIN |
| sphinx-doc__sphinx-8265.S.claude-opus-5-5.1 | U | tests/test_pycode_ast.py | 56 | replace | the file is UNCERTAIN |
| sphinx-doc__sphinx-8265.S.claude-opus-5-5.1 | U | sphinx/pycode/ast.py | None | terse | a --terse receipt names no hunk address |
| sphinx-doc__sphinx-8265.S.claude-opus-5-5.1 | U | tests/test_pycode_ast.py | None | terse | a --terse receipt names no hunk address |
| sympy__sympy-24661.S.claude-opus-5-5.1 | U | sympy/parsing/tests/test_sympy_parser.py | 361 | replace | the file is UNCERTAIN |
| astropy__astropy-14539.B.claude-sonnet-5-5.1 | U | astropy/io/fits/diff.py | 1452 | replace | the file is UNCERTAIN |
| astropy__astropy-14539.S.claude-sonnet-5-5.1 | U | astropy/io/fits/diff.py | 1452 | replace | the file is UNCERTAIN |
| django__django-11451.B.claude-sonnet-5-5.1 | U | django/contrib/auth/backends.py | 41 | insert-after | write calls and receipts do not pair |
| django__django-11532.B.claude-sonnet-5-5.1 | U | django/core/mail/message.py | 260 | replace | the file is UNCERTAIN |
| django__django-11532.B.claude-sonnet-5-5.1 | U | django/core/mail/message.py | 260 | replace | the file is UNCERTAIN |
| django__django-11532.S.claude-sonnet-5-5.1 | U | django/core/mail/message.py | 260 | replace | the file is UNCERTAIN |
| django__django-11555.B.claude-sonnet-5-5.1 | U | django/db/models/sql/compiler.py | 724 | replace | the file is UNCERTAIN |
| django__django-11555.S.claude-sonnet-5-5.1 | U | django/db/models/sql/compiler.py | 724 | replace | the file is UNCERTAIN |
| django__django-13807.B.claude-sonnet-5-5.1 | U | django/db/backends/sqlite3/base.py | 330 | replace | the file is UNCERTAIN |
| django__django-13807.B.claude-sonnet-5-5.1 | U | django/db/backends/sqlite3/base.py | 336 | replace | the file is UNCERTAIN |
| django__django-13807.S.claude-sonnet-5-5.1 | U | django/db/backends/sqlite3/base.py | 330 | replace | the file is UNCERTAIN |
| django__django-13807.S.claude-sonnet-5-5.1 | U | django/db/backends/sqlite3/base.py | 336 | replace | the file is UNCERTAIN |
| django__django-14434.S.claude-sonnet-5-5.1 | U | django/db/backends/base/schema.py | 1254 | replace | the file is UNCERTAIN |
| django__django-16116.B.claude-sonnet-5-5.1 | U | django/core/management/commands/makemigrations.py | 251 | replace | the file is UNCERTAIN |
| django__django-16116.S.claude-sonnet-5-5.1 | U | django/core/management/commands/makemigrations.py | 250 | replace | the file is UNCERTAIN |
| django__django-16642.B.claude-sonnet-5-5.1 | U | django/http/response.py | 614 | insert-after | the file is UNCERTAIN |
| matplotlib__matplotlib-25775.B.claude-sonnet-5-5.1 | U | backends/backend_agg.py | 209 | replace | write calls and receipts do not pair |
| matplotlib__matplotlib-25775.B.claude-sonnet-5-5.1 | U | backends/backend_cairo.py | 208 | replace | write calls and receipts do not pair |
| pytest-dev__pytest-5631.S.claude-sonnet-5-5.1 | U | src/_pytest/compat.py | 70 | replace | the file is UNCERTAIN |
| scikit-learn__scikit-learn-25747.B.claude-sonnet-5-5.1 | U | sklearn/utils/_set_output.py | 58 | replace | the file is UNCERTAIN |
| sphinx-doc__sphinx-8265.S.claude-sonnet-5-5.1 | U | sphinx/pycode/ast.py | 176 | replace | the file is UNCERTAIN |
| sympy__sympy-18199.B.claude-sonnet-5-5.1 | U | sympy/ntheory/residue_ntheory.py | 772 | insert-after | the file is UNCERTAIN |
| sympy__sympy-24661.S.claude-sonnet-5-5.1 | U | sympy/parsing/sympy_parser.py | 1189 | replace | the file is UNCERTAIN |
| sympy__sympy-24661.S.claude-sonnet-5-5.1 | U | sympy/parsing/sympy_parser.py | 1111 | replace | the file is UNCERTAIN |
