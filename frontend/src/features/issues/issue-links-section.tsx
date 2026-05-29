import { type FormEvent } from "react";

import { FormError } from "../../components/form-feedback";
import {
  type Issue,
  type IssueLink,
  type IssueLinkType,
} from "../../lib/api-types";
import { issueTypeLabels, statusLabel } from "../../lib/issue-model";

type IssueLinksSectionProps = {
  availableIssues: Issue[];
  canCreateLink: boolean;
  deletingLinkIds: string[];
  formError: string;
  isCreatingLink: boolean;
  isLoadingLinks: boolean;
  issue: Issue;
  linkTargetIssueId: string;
  linkType: IssueLinkType;
  links: IssueLink[];
  linksError: string;
  onCreateLink: (event: FormEvent<HTMLFormElement>) => void;
  onDeleteLink: (link: IssueLink) => void;
  onOpenIssue: (issueId: string) => void;
  onTargetIssueChange: (issueId: string) => void;
  onTypeChange: (linkType: IssueLinkType) => void;
};

const linkTypeLabels: Record<IssueLinkType, string> = {
  blocks: "blocks",
  relates: "relates",
};

export function IssueLinksSection({
  availableIssues,
  canCreateLink,
  deletingLinkIds,
  formError,
  isCreatingLink,
  isLoadingLinks,
  issue,
  linkTargetIssueId,
  linkType,
  links,
  linksError,
  onCreateLink,
  onDeleteLink,
  onOpenIssue,
  onTargetIssueChange,
  onTypeChange,
}: IssueLinksSectionProps) {
  return (
    <section className="linked-issues-section">
      <div className="comments-header">
        <div>
          <p className="eyebrow">Links</p>
          <h3>Linked issues</h3>
        </div>
        {isLoadingLinks ? <span className="muted">Loading</span> : null}
      </div>

      {linksError ? <FormError message={linksError} /> : null}

      {links.length > 0 ? (
        <div className="linked-issue-list">
          {links.map((link) => {
            const linkedIssue =
              link.source_issue_id === issue.id
                ? link.target_issue
                : link.source_issue;
            const relation = linkRelationLabel(issue.id, link);

            return (
              <article className="linked-issue-card" key={link.id}>
                <button
                  className="hierarchy-issue-button"
                  onClick={() => onOpenIssue(linkedIssue.id)}
                  type="button"
                >
                  <strong>
                    {linkedIssue.issue_key} · {linkedIssue.title}
                  </strong>
                  <small>
                    {issueTypeLabels[linkedIssue.issue_type]} ·{" "}
                    {statusLabel(linkedIssue.status)}
                  </small>
                </button>

                <div className="linked-issue-actions">
                  <span className="link-type-pill">{relation}</span>
                  <button
                    className="ghost-button danger-button"
                    disabled={deletingLinkIds.includes(link.id)}
                    onClick={() => onDeleteLink(link)}
                    type="button"
                  >
                    {deletingLinkIds.includes(link.id) ? "Removing" : "Remove"}
                  </button>
                </div>
              </article>
            );
          })}
        </div>
      ) : (
        <div className="linked-issues-empty">
          <strong>No linked issues yet</strong>
          <span className="muted">
            Add a blocks or relates connection when another issue depends on
            this work.
          </span>
        </div>
      )}

      <form className="issue-link-form" onSubmit={onCreateLink}>
        <label>
          <span>Relationship</span>
          <select
            onChange={(event) =>
              onTypeChange(event.target.value as IssueLinkType)
            }
            value={linkType}
          >
            {Object.entries(linkTypeLabels).map(([value, label]) => (
              <option key={value} value={value}>
                {label}
              </option>
            ))}
          </select>
        </label>

        <label>
          <span>Target issue</span>
          <select
            onChange={(event) => onTargetIssueChange(event.target.value)}
            value={linkTargetIssueId}
          >
            <option value="">Choose issue</option>
            {availableIssues.map((availableIssue) => (
              <option key={availableIssue.id} value={availableIssue.id}>
                {availableIssue.issue_key} · {availableIssue.title}
              </option>
            ))}
          </select>
        </label>

        <button disabled={!canCreateLink} type="submit">
          {isCreatingLink ? "Adding" : "Add link"}
        </button>

        {formError ? <FormError message={formError} /> : null}
      </form>
    </section>
  );
}

function linkRelationLabel(issueId: string, link: IssueLink) {
  if (link.link_type === "relates") {
    return "relates";
  }

  return link.source_issue_id === issueId ? "blocks" : "is blocked by";
}
