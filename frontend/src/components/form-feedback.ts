import { createElement } from "react";

type FormFeedbackProps = {
  message: string;
};

export function FormError({ message }: FormFeedbackProps) {
  if (!message) {
    return null;
  }

  return createElement(
    "p",
    {
      className: "form-error",
      role: "alert",
    },
    message,
  );
}

