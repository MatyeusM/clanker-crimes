import { render } from "preact";
import { LocationProvider, Router, Route } from "preact-iso";

import { Header } from "./components/Header.jsx";
import { NotFound } from "./pages/_404.jsx";
import { Home } from "./pages/Home/index.jsx";

import "./style.css";

export function App() {
  return (
    <LocationProvider>
      <Header />
      <main>
        <Router>
          <Route
            path="/"
            component={Home}
          />
          <Route
            default
            component={NotFound}
          />
        </Router>
      </main>
    </LocationProvider>
  );
}

const root = document.getElementById("app");
if (!root) {
  throw new Error("Missing #app element");
}

render(<App />, root);
