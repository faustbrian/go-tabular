# Upstream review history

## 2026-09-03

The current normative payloads remain byte-identical to the manifest pins:
MS-CFB 12.0 is
`9d0d61e34495347ee32f3de5b06f2d59953cc60607ea72605d4162d21a34863f`,
MS-XLS 12.2 is
`5711b0c9d3ca5821d4a7649c6e1abd0762228f7909311aeb86e90d72f64055b2`,
MS-OI29500 25.0 is
`b297063cce0ac79d10a8efd382b0f90f3b9fd6615fac7f01c88f0288e5fa7372`,
and PKWARE APPNOTE 6.3.10 is
`0b993022a7d320a0bf704e6980bea36fafd17a6066ab994db0a0c16278a50cd6`.

| Authorities | Previous SHA-256 | Current SHA-256 | Semantic delta | Decisions | Conclusion |
| --- | --- | --- | --- | --- | --- |
| `ms-cfb-source`, `ms-cfb-releases` | `de5dca29431f4da7cd51d3d0c6424297c7c0af3cad563a327417504bbce41504` | `e76e8683d196eedc9bf5f768135a3d65bf2422ce80773813b45667e94dae0939` | Microsoft landing-page presentation changed; the pinned MS-CFB 12.0 PDF did not. | TABULAR-DEC-004 | Behavior-neutral; retain the decision and conformance binding. |
| `ms-xls-source`, `ms-xls-releases` | `fa50f8817a8f9bed70afd0bf28d71cfee02a62a8512e1ac18df2b1402a28e2ba` | `ef1903dc50bc6b3ab32a7f7cc5ea6bdde3fda06a1a439469803d06acdc482f4e` | Microsoft landing-page presentation changed; the pinned MS-XLS 12.2 PDF did not. | TABULAR-DEC-005 | Behavior-neutral; retain the decision and conformance binding. |
| `ooxml-source`, `ooxml-releases` | `cac1240e3e9900bc5830eab652830dc9013d8360da7753ddc8c0a0b89721cedb` | `ff75e7fa73f1ad7d9f1ecdf2fe54999299c50e29d405849c80f6960bef35c183` | Microsoft landing-page presentation changed; the pinned MS-OI29500 25.0 PDF and ECMA-376 edition did not. | TABULAR-DEC-006, TABULAR-DEC-007 | Behavior-neutral; retain both decisions and conformance bindings. |
| `zip-releases` | `c200a6e4762e2313db73369fef02a658f51ce51d104406b1aa371bdcbb0dd8b9` | `a9edf1ec360617fdc5d049dd63b709d6ddc48510c10380d5d1a2d3d691518eaf` | PKWARE landing-page presentation changed; the pinned APPNOTE 6.3.10 payload did not. | TABULAR-DEC-008 | Behavior-neutral; retain the decision and conformance binding. |

The current landing-page digests were retrieved three times on 2026-09-03
with the specification checker's request profile and were stable. They are
monitoring signals only; mutable page chrome is not normative format content.

## 2026-09-04

The Microsoft publication pages changed again, but each still identifies the
same published version and links the same byte-identical normative payload:
MS-CFB 12.0 (2024-04-23) is
`9d0d61e34495347ee32f3de5b06f2d59953cc60607ea72605d4162d21a34863f`,
MS-XLS 12.2 (2025-08-19) is
`5711b0c9d3ca5821d4a7649c6e1abd0762228f7909311aeb86e90d72f64055b2`,
and MS-OI29500 25.0 (2026-08-18) is
`b297063cce0ac79d10a8efd382b0f90f3b9fd6615fac7f01c88f0288e5fa7372`.

| Authorities | Previous SHA-256 | Current SHA-256 | Semantic delta | Decisions | Conclusion |
| --- | --- | --- | --- | --- | --- |
| `ms-cfb-source`, `ms-cfb-releases` | `e76e8683d196eedc9bf5f768135a3d65bf2422ce80773813b45667e94dae0939` | `057bd660ef7946495453559e60507dcd0932bf161c635fc6d782547458972569` | Microsoft landing-page presentation changed; the published date and pinned MS-CFB 12.0 PDF did not. | TABULAR-DEC-004 | Behavior-neutral; retain the decision and conformance binding. |
| `ms-xls-source`, `ms-xls-releases` | `ef1903dc50bc6b3ab32a7f7cc5ea6bdde3fda06a1a439469803d06acdc482f4e` | `4cd1395ec7744fff92ba03c53881ebe5f6ca07ec045fe2e02f4a48b2dda4b23a` | Microsoft landing-page presentation changed; the published date and pinned MS-XLS 12.2 PDF did not. | TABULAR-DEC-005 | Behavior-neutral; retain the decision and conformance binding. |
| `ooxml-source`, `ooxml-releases` | `ff75e7fa73f1ad7d9f1ecdf2fe54999299c50e29d405849c80f6960bef35c183` | `e34908432b10f3c71cebd683edd17be844d888b5864752ecc3a01464adfde3fb` | Microsoft landing-page presentation changed; the published date and pinned MS-OI29500 25.0 PDF did not. | TABULAR-DEC-006, TABULAR-DEC-007 | Behavior-neutral; retain both decisions and conformance bindings. |
| `zip-releases` | `a9edf1ec360617fdc5d049dd63b709d6ddc48510c10380d5d1a2d3d691518eaf` | `0b993022a7d320a0bf704e6980bea36fafd17a6066ab994db0a0c16278a50cd6` | Replace the mutable product landing page with PKWARE's current general-release APPNOTE payload; PKWARE's archive still ends at 6.3.10 and the pinned bytes did not change. | TABULAR-DEC-008 | Behavior-neutral; retain the decision and conformance binding. |

Each landing-page digest was retrieved three times with the specification
checker's request profile and was stable. The repeated changes remain
monitoring signals only; no normative content or package behavior changed.
PKWARE's publication policy keeps the current general release at the stable
`APPNOTE.TXT` path, so that payload now detects both source and release changes
without depending on unrelated product-page presentation.
