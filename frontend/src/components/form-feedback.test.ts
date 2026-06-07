import assert from "node:assert/strict";
import { test } from "vitest";
import { renderToStaticMarkup } from "react-dom/server";
import { FormError } from "./form-feedback.ts";

test("renders form errors as accessible alerts", () => {
  const markup = renderToStaticMarkup(FormError({ message: "Project name is required." }));

  assert.match(markup, /role="alert"/);
  assert.match(markup, /class="form-error"/);
  assert.match(markup, /Project name is required\./);
});

test("does not render empty form errors", () => {
  const markup = renderToStaticMarkup(FormError({ message: "" }));

  assert.equal(markup, "");
});

