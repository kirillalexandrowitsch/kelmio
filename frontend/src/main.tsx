import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import "@fontsource-variable/manrope";
import "@fontsource-variable/space-grotesk";
import { App } from "./App";
import "./styles/tokens.css";
import "./styles/reset.css";
import "./styles/primitives.css";
import "./styles.css";
import "./styles/auth.css";
import "./styles/shell.css";
import "./styles/workspace.css";

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <App />
  </StrictMode>,
);
