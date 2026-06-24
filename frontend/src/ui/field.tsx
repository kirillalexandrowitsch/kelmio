import {
  cloneElement,
  forwardRef,
  type InputHTMLAttributes,
  isValidElement,
  type ReactElement,
  type SelectHTMLAttributes,
  type TextareaHTMLAttributes,
  useId,
} from "react";

type FieldControlProps = {
  "aria-describedby"?: string;
  "aria-labelledby"?: string;
  id?: string;
};

type FieldProps = {
  children: ReactElement<FieldControlProps>;
  className?: string;
  hint?: string;
  label: string;
};

export function Field({ children, className = "", hint, label }: FieldProps) {
  const generatedId = useId();
  const controlId = children.props.id ?? `${generatedId}-control`;
  const labelId = `${generatedId}-label`;
  const hintId = `${generatedId}-hint`;
  const control = isValidElement<FieldControlProps>(children)
    ? cloneElement(children, {
        "aria-describedby": hint ? hintId : children.props["aria-describedby"],
        "aria-labelledby": labelId,
        id: controlId,
      })
    : children;

  return (
    <div className={`ui-field ${className}`.trim()}>
      <label className="ui-field-label" htmlFor={controlId} id={labelId}>
        {label}
      </label>
      {control}
      {hint ? (
        <span className="ui-field-hint" id={hintId}>
          {hint}
        </span>
      ) : null}
    </div>
  );
}

export const Input = forwardRef<HTMLInputElement, InputHTMLAttributes<HTMLInputElement>>(
  function Input({ className = "", ...props }, ref) {
    return <input className={`ui-input ${className}`.trim()} ref={ref} {...props} />;
  },
);

export const Select = forwardRef<HTMLSelectElement, SelectHTMLAttributes<HTMLSelectElement>>(
  function Select({ className = "", ...props }, ref) {
    return <select className={`ui-select ${className}`.trim()} ref={ref} {...props} />;
  },
);

export const Textarea = forwardRef<
  HTMLTextAreaElement,
  TextareaHTMLAttributes<HTMLTextAreaElement>
>(function Textarea({ className = "", ...props }, ref) {
  return <textarea className={`ui-textarea ${className}`.trim()} ref={ref} {...props} />;
});
