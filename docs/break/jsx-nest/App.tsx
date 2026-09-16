// Attack fixture for the JSX nest class (BACKLOG / dangling spec UC-5).
// The extra wrapper is balanced, so tsc stays green, but #inner's parent is
// #accidental-wrapper instead of #intended-parent. Do not "fix" this file.
export function App() {
  return (
    <div id="outer">
      <section id="intended-parent">
        <div id="accidental-wrapper">
          <p id="inner">hello</p>
        </div>
      </section>
    </div>
  );
}
