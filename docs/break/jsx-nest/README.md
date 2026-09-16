# JSX nest probe

Deliberately wrong-DOM. Do not tidy `App.tsx`.

**Intended parent of `#inner`:** `#intended-parent` (the `<section>`).

**Actual parent after a correct parse:** `#accidental-wrapper`.

That is the class: a balanced extra wrapper (the YAML-indent analogue). `tsc` is
green because the tree is valid TSX. The DOM is wrong because `#inner` is not
where the author meant.

## Typecheck

`npx --yes --package typescript tsc --noEmit --jsx react-jsx --strict App.tsx` is red
here without `react` on disk (`TS2875` jsx-runtime). The class is not that miss.
Copy `App.tsx` into a throwaway npm tree with `typescript`, `react`, `@types/react`:

```bash
npx tsc --noEmit --jsx react-jsx --strict --esModuleInterop App.tsx
```

Expected: exit 0. Observed 2026-09-16: 0.

## DOM

`react-dom/server` `renderToStaticMarkup` of `App` (same throwaway tree, `esbuild`
bundle) on 2026-09-16:

```html
<div id="outer"><section id="intended-parent"><div id="accidental-wrapper"><p id="inner">hello</p></div></section></div>
```

`#inner`'s parent is `#accidental-wrapper`, not `#intended-parent`.

## Why it stays

A write-time parser is ADR-048 Out of Scope. Reproduction is evidence for a
later quote, not a parser.
