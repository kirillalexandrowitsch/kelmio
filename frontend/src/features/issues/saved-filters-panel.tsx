import { FormEvent } from "react";
import { FormError } from "../../components/form-feedback";
import { type SavedFilter } from "../../lib/api-types";
import { savedIssueFilterSummary } from "../../lib/issue-model";
import { Button, Field, Input } from "../../ui";

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
    <section className="kl-card kl-saved-filters" aria-label="Saved issue filters">
      <div className="kl-section-head">
        <div>
          <p className="kl-eyebrow">Saved views</p>
          <h3>Issue filters</h3>
        </div>
        {isLoadingSavedFilters ? <span className="kl-muted">Loading</span> : null}
      </div>

      <form className="kl-saved-filters__form" onSubmit={onCreateSavedFilter}>
        <Field label="View name" htmlFor="saved-filter-name">
          <Input
            id="saved-filter-name"
            onChange={(event) => onSavedFilterNameChange(event.target.value)}
            placeholder="My todo bugs"
            value={savedFilterName}
          />
        </Field>
        <Button variant="secondary" disabled={isCreatingSavedFilter} type="submit">
          {isCreatingSavedFilter ? "Saving" : "Save current view"}
        </Button>
      </form>

      <FormError message={savedFilterFormError} />
      <FormError message={savedFiltersError} />

      {savedFilters.length > 0 ? (
        <div className="kl-saved-filters__list">
          {savedFilters.map((savedFilter) => {
            const isUpdating = updatingSavedFilterIds.includes(savedFilter.id);
            const isDeleting = deletingSavedFilterIds.includes(savedFilter.id);
            const isRenaming = renameSavedFilterId === savedFilter.id;

            return (
              <article className="kl-saved-filter" key={savedFilter.id}>
                {isRenaming ? (
                  <form
                    className="kl-saved-filter__rename"
                    onSubmit={(event) => {
                      event.preventDefault();
                      onRenameSavedFilter(savedFilter);
                    }}
                  >
                    <Input
                      onChange={(event) =>
                        onRenameSavedFilterNameChange(event.target.value)
                      }
                      value={renameSavedFilterName}
                    />
                    <Button size="sm" variant="primary" disabled={isUpdating} type="submit">
                      Save name
                    </Button>
                    <Button size="sm" variant="ghost" onClick={onCancelRenameSavedFilter}>
                      Cancel
                    </Button>
                  </form>
                ) : (
                  <>
                    <div className="kl-saved-filter__info">
                      <h4>{savedFilter.name}</h4>
                      <p>
                        {savedIssueFilterSummary(
                          savedFilter.filters,
                          workflowStatusNamesById,
                        ).join(" · ")}
                      </p>
                    </div>
                    <div className="kl-saved-filter__actions">
                      <Button
                        size="sm"
                        variant="secondary"
                        onClick={() => onApplySavedFilter(savedFilter)}
                      >
                        Apply
                      </Button>
                      <Button
                        size="sm"
                        variant="ghost"
                        disabled={isUpdating}
                        onClick={() => onUpdateSavedFilter(savedFilter)}
                      >
                        {isUpdating ? "Updating" : "Update"}
                      </Button>
                      <Button
                        size="sm"
                        variant="ghost"
                        onClick={() => onStartRenameSavedFilter(savedFilter)}
                      >
                        Rename
                      </Button>
                      <Button
                        size="sm"
                        variant="danger"
                        disabled={isDeleting}
                        onClick={() => onDeleteSavedFilter(savedFilter)}
                      >
                        {isDeleting ? "Deleting" : "Delete"}
                      </Button>
                    </div>
                  </>
                )}
              </article>
            );
          })}
        </div>
      ) : (
        <p className="kl-muted">No saved filters yet.</p>
      )}
    </section>
  );
}
