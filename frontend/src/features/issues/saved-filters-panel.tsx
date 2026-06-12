import { FormEvent } from "react";
import { FormError } from "../../components/form-feedback";
import { type SavedFilter } from "../../lib/api-types";
import { savedIssueFilterSummary } from "../../lib/issue-model";

type SavedFiltersPanelProps = {
  deletingSavedFilterIds: string[];
  isCreatingSavedFilter: boolean;
  isLoadingSavedFilters: boolean;
  onApplySavedFilter: (savedFilter: SavedFilter) => void;
  onCancelRenameSavedFilter: () => void;
  onCreateSavedFilter: (event: FormEvent<HTMLFormElement>) => void;
  onDeleteSavedFilter: (savedFilter: SavedFilter) => void;
  onRenameSavedFilter: (savedFilter: SavedFilter) => void;
  onRenameSavedFilterNameChange: (value: string) => void;
  onSavedFilterNameChange: (value: string) => void;
  onStartRenameSavedFilter: (savedFilter: SavedFilter) => void;
  onUpdateSavedFilter: (savedFilter: SavedFilter) => void;
  renameSavedFilterId: string;
  renameSavedFilterName: string;
  savedFilterFormError: string;
  savedFilterName: string;
  savedFilters: SavedFilter[];
  savedFiltersError: string;
  updatingSavedFilterIds: string[];
  workflowStatusNamesById: Record<string, string>;
};

export function SavedFiltersPanel({
  deletingSavedFilterIds,
  isCreatingSavedFilter,
  isLoadingSavedFilters,
  onApplySavedFilter,
  onCancelRenameSavedFilter,
  onCreateSavedFilter,
  onDeleteSavedFilter,
  onRenameSavedFilter,
  onRenameSavedFilterNameChange,
  onSavedFilterNameChange,
  onStartRenameSavedFilter,
  onUpdateSavedFilter,
  renameSavedFilterId,
  renameSavedFilterName,
  savedFilterFormError,
  savedFilterName,
  savedFilters,
  savedFiltersError,
  updatingSavedFilterIds,
  workflowStatusNamesById,
}: SavedFiltersPanelProps) {
  return (
    <section className="saved-filters-panel" aria-label="Saved issue filters">
      <div className="saved-filters-header">
        <div>
          <p className="eyebrow">Saved views</p>
          <h3>Issue filters</h3>
        </div>
        {isLoadingSavedFilters ? <span className="muted">Loading</span> : null}
      </div>

      <form className="saved-filter-form" onSubmit={onCreateSavedFilter}>
        <label>
          <span>View name</span>
          <input
            onChange={(event) => onSavedFilterNameChange(event.target.value)}
            placeholder="My todo bugs"
            value={savedFilterName}
          />
        </label>
        <button
          className="small-button"
          disabled={isCreatingSavedFilter}
          type="submit"
        >
          {isCreatingSavedFilter ? "Saving" : "Save current view"}
        </button>
      </form>

      <FormError message={savedFilterFormError} />
      <FormError message={savedFiltersError} />

      {savedFilters.length > 0 ? (
        <div className="saved-filter-list">
          {savedFilters.map((savedFilter) => {
            const isUpdating = updatingSavedFilterIds.includes(savedFilter.id);
            const isDeleting = deletingSavedFilterIds.includes(savedFilter.id);
            const isRenaming = renameSavedFilterId === savedFilter.id;

            return (
              <article className="saved-filter-card" key={savedFilter.id}>
                {isRenaming ? (
                  <form
                    className="saved-filter-rename"
                    onSubmit={(event) => {
                      event.preventDefault();
                      onRenameSavedFilter(savedFilter);
                    }}
                  >
                    <input
                      onChange={(event) =>
                        onRenameSavedFilterNameChange(event.target.value)
                      }
                      value={renameSavedFilterName}
                    />
                    <button
                      className="small-button"
                      disabled={isUpdating}
                      type="submit"
                    >
                      Save name
                    </button>
                    <button
                      className="small-button"
                      onClick={onCancelRenameSavedFilter}
                      type="button"
                    >
                      Cancel
                    </button>
                  </form>
                ) : (
                  <>
                    <div>
                      <h4>{savedFilter.name}</h4>
                      <p>
                        {savedIssueFilterSummary(
                          savedFilter.filters,
                          workflowStatusNamesById,
                        ).join(" · ")}
                      </p>
                    </div>
                    <div className="saved-filter-actions">
                      <button
                        className="small-button"
                        onClick={() => onApplySavedFilter(savedFilter)}
                        type="button"
                      >
                        Apply
                      </button>
                      <button
                        className="small-button"
                        disabled={isUpdating}
                        onClick={() => onUpdateSavedFilter(savedFilter)}
                        type="button"
                      >
                        {isUpdating ? "Updating" : "Update"}
                      </button>
                      <button
                        className="small-button"
                        onClick={() => onStartRenameSavedFilter(savedFilter)}
                        type="button"
                      >
                        Rename
                      </button>
                      <button
                        className="small-button danger-button"
                        disabled={isDeleting}
                        onClick={() => onDeleteSavedFilter(savedFilter)}
                        type="button"
                      >
                        {isDeleting ? "Deleting" : "Delete"}
                      </button>
                    </div>
                  </>
                )}
              </article>
            );
          })}
        </div>
      ) : (
        <p className="saved-filter-empty">No saved filters yet.</p>
      )}
    </section>
  );
}
