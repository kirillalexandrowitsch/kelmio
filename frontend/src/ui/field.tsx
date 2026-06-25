import {
  type InputHTMLAttributes,
  type ReactNode,
  type SelectHTMLAttributes,
  type TextareaHTMLAttributes,
} from "react";

type FieldProps = {
  label: ReactNode;
  htmlFor?: string;
  hint?: ReactNode;
  error?: ReactNode;
  children: ReactNode;
  className?: string;
};

export function Field({
  label,
  htmlFor,
  hint,
  error,
  children,
  className,
}: FieldProps) {
  const classes = ["kl-field", error ? "kl-field--invalid" : null, className]
    .filter(Boolean)
    .join(" ");

  return (
    <div className={classes}>
      <label className="kl-field__label" htmlFor={htmlFor}>
        {label}
      </label>
      {children}
      {hint && !error ? <span className="kl-field__hint">{hint}</span> : null}
      {error ? <span className="kl-field__error">{error}</span> : null}
    </div>
  );
}

export function Input({
  className,
  ...rest
}: InputHTMLAttributes<HTMLInputElement>) {
  return (
    <input className={["kl-input", className].filter(Boolean).join(" ")} {...rest} />
  );
}

export function TextArea({
  className,
  ...rest
}: TextareaHTMLAttributes<HTMLTextAreaElement>) {
  return (
    <textarea
      className={["kl-textarea", className].filter(Boolean).join(" ")}
      {...rest}
    />
  );
}

export function Select({
  className,
  children,
  ...rest
}: SelectHTMLAttributes<HTMLSelectElement>) {
  return (
    <select
      className={["kl-select", className].filter(Boolean).join(" ")}
      {...rest}
    >
      {children}
    </select>
  );
}
