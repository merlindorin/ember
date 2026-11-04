# Taffy fixtures

From github.com/DioxusLabs/taffy at c6421c1c (MIT, `../../LICENSE-taffy`): `tests/xml/<suite>/*_ltr.xml`, one file per suite. Taffy's gentest lays every tree out in Chrome and writes the boxes as `<expectations>`.

Left out, being Taffy extensions or CSS that ui doesn't support (decision 0049): `flex-wrap: balance` and `flex-line-count`, floats and `clear`, `display: flow-root`, `-webkit-` alignment, vertical writing modes, right-to-left, and the `float` and `contain` suites. The `direction="ltr"` attribute and the `_ltr` name suffix are dropped.

Text leaves are Ahem: 10×10 glyphs, broken only at U+200B.

To regenerate from a Taffy checkout:

```python
import os, re
OUT_OF_SCOPE = ['balance', 'flex-line-count', 'float=', 'clear=', 'flow-root', '-webkit-', 'writing-mode', 'direction="rtl"']
for suite in ["flex", "grid", "block", "leaf", "blockflex", "blockgrid", "gridflex"]:
    d = f"taffy/tests/xml/{suite}"
    out = [f'<fixtures suite="{suite}" source="github.com/DioxusLabs/taffy@REV tests/xml/{suite}, left-to-right only">']
    for n in sorted(n for n in os.listdir(d) if n.endswith("_ltr.xml")):
        x = open(os.path.join(d, n), encoding="utf-8").read().strip().replace(' direction="ltr"', '')
        x = re.sub(r'(<test name="[^"]*)_ltr"', r'\1"', x, count=1)
        if not any(k in x for k in OUT_OF_SCOPE):
            out.append(x)
    open(f"{suite}.xml", "w", encoding="utf-8").write("\n".join(out + ["</fixtures>\n"]))
```
