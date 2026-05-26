import { type FormEvent } from "react";

import { FormError } from "../../components/form-feedback";
import { type Label } from "../../lib/api-types";

type LabelsSectionProps = {
  canCreateLabel: boolean;
  deletingLabelIds: string[];
  isActive: boolean;
  isCreatingLabel: boolean;
  isLoadingLabels: boolean;
  labelColor: string;
  labelName: string;
  labels: Label[];
  labelsError: string;
  onColorChange: (value: string) => void;
  onCreateLabel: (event: FormEvent<HTMLFormElement>) => void;
  onDeleteLabel: (label: Label) => void;
  onNameChange: (value: string) => void;
};

export function LabelsSection({
  canCreateLabel,
  deletingLabelIds,
  isActive,
  isCreatingLabel,
  isLoadingLabels,
  labelColor,
  labelName,
  labels,
  labelsError,
  onColorChange,
  onCreateLabel,
  onDeleteLabel,
  onNameChange,
}: LabelsSectionProps) {
  return (
    <section
      className="labels-panel"
      aria-label="Labels"
      hidden={!isActive}
    >
      <header className="section-header">
        <div>
          <p className="eyebrow">Labels</p>
          <h2>Workspace labels</h2>
        </div>
        {isLoadingLabels ? <span className="muted">Loading</span> : null}
      </header>

      <FormError message={labelsError} />

      {labels.length > 0 ? (
        <div className="label-list">
          {labels.map((label) => {
            const isDeletingLabel = deletingLabelIds.includes(label.id);

            return (
              <div className="label-management-row" key={label.id}>
                <span
                  className="label-chip"
                  style={{
                    backgroundColor: `${label.color}1a`,
                    borderColor: label.color,
                  }}
                >
                  {label.name}
                </span>
                <button
                  className="small-button danger-button"
                  disabled={isDeletingLabel}
                  onClick={() => {
                    onDeleteLabel(label);
                  }}
                  type="button"
                >
                  {isDeletingLabel ? "Deleting..." : "Delete"}
                </button>
              </div>
            );
          })}
        </div>
      ) : (
        <div className="labels-empty">No labels yet</div>
      )}

      <form className="label-form" onSubmit={onCreateLabel}>
        <label>
          <span>Name</span>
          <input
            maxLength={40}
            onChange={(event) => onNameChange(event.target.value)}
            placeholder="frontend"
            value={labelName}
          />
        </label>
        <label>
          <span>Color</span>
          <input
            onChange={(event) => onColorChange(event.target.value)}
            type="color"
            value={labelColor}
          />
        </label>
        <button disabled={!canCreateLabel} type="submit">
          {isCreatingLabel ? "Creating..." : "Create label"}
        </button>
      </form>
    </section>
  );
}
