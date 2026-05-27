import { type FormEvent } from "react";

import { FormError } from "../../components/form-feedback";
import { type CurrentUser, type IssueComment } from "../../lib/api-types";
import { formatDateTime } from "../../lib/formatting";
import { hasText } from "../../lib/validation";

type IssueCommentsSectionProps = {
  canCreateComment: boolean;
  commentBody: string;
  comments: IssueComment[];
  commentsError: string;
  currentUser: CurrentUser;
  deletingCommentIds: string[];
  editCommentBody: string;
  editingCommentId: string;
  isCreatingComment: boolean;
  isLoadingComments: boolean;
  onCancelEditingComment: () => void;
  onCommentBodyChange: (value: string) => void;
  onCreateComment: (event: FormEvent<HTMLFormElement>) => void;
  onDeleteComment: (comment: IssueComment) => void;
  onEditCommentBodyChange: (value: string) => void;
  onStartEditingComment: (comment: IssueComment) => void;
  onUpdateComment: (
    event: FormEvent<HTMLFormElement>,
    comment: IssueComment,
  ) => void;
  updatingCommentIds: string[];
};

export function IssueCommentsSection({
  canCreateComment,
  commentBody,
  comments,
  commentsError,
  currentUser,
  deletingCommentIds,
  editCommentBody,
  editingCommentId,
  isCreatingComment,
  isLoadingComments,
  onCancelEditingComment,
  onCommentBodyChange,
  onCreateComment,
  onDeleteComment,
  onEditCommentBodyChange,
  onStartEditingComment,
  onUpdateComment,
  updatingCommentIds,
}: IssueCommentsSectionProps) {
  return (
    <section className="comments-section" aria-label="Issue comments">
      <header className="comments-header">
        <div>
          <p className="eyebrow">Comments</p>
          <h3>{comments.length}</h3>
        </div>
        {isLoadingComments ? (
          <span className="muted">Loading comments</span>
        ) : null}
      </header>

      {commentsError ? <FormError message={commentsError} /> : null}

      {comments.length > 0 ? (
        <div className="comment-list">
          {comments.map((comment) => {
            const isEditingComment = editingCommentId === comment.id;
            const isUpdatingComment = updatingCommentIds.includes(comment.id);
            const isDeletingComment = deletingCommentIds.includes(comment.id);
            const canEditComment =
              comment.author_id === currentUser.id ||
              currentUser.workspace.role === "admin";
            const wasEdited = comment.updated_at !== comment.created_at;

            return (
              <article className="comment-card" key={comment.id}>
                <header>
                  <div className="comment-author">
                    <strong>{comment.author_display_name}</strong>
                    <span>
                      {formatDateTime(comment.created_at)}
                      {wasEdited
                        ? ` · edited ${formatDateTime(comment.updated_at)}`
                        : ""}
                    </span>
                  </div>
                  {canEditComment ? (
                    <div className="comment-actions">
                      <button
                        className="small-button"
                        disabled={isUpdatingComment || isDeletingComment}
                        onClick={() => {
                          if (isEditingComment) {
                            onCancelEditingComment();
                          } else {
                            onStartEditingComment(comment);
                          }
                        }}
                        type="button"
                      >
                        {isEditingComment ? "Cancel" : "Edit"}
                      </button>
                      <button
                        className="small-button danger-button"
                        disabled={isUpdatingComment || isDeletingComment}
                        onClick={() => onDeleteComment(comment)}
                        type="button"
                      >
                        {isDeletingComment ? "Deleting..." : "Delete"}
                      </button>
                    </div>
                  ) : null}
                </header>

                {isEditingComment ? (
                  <form
                    className="comment-edit-form"
                    onSubmit={(event) => onUpdateComment(event, comment)}
                  >
                    <textarea
                      maxLength={4000}
                      onChange={(event) =>
                        onEditCommentBodyChange(event.target.value)
                      }
                      rows={3}
                      value={editCommentBody}
                    />
                    <div className="form-actions">
                      <button
                        disabled={isUpdatingComment || !hasText(editCommentBody)}
                        type="submit"
                      >
                        {isUpdatingComment ? "Saving..." : "Save"}
                      </button>
                      <button
                        className="ghost-button"
                        disabled={isUpdatingComment}
                        onClick={onCancelEditingComment}
                        type="button"
                      >
                        Cancel
                      </button>
                    </div>
                  </form>
                ) : (
                  <p>{comment.body}</p>
                )}
              </article>
            );
          })}
        </div>
      ) : (
        <div className="comments-empty">No comments yet</div>
      )}

      <form className="comment-form" onSubmit={onCreateComment}>
        <label>
          <span>Add comment</span>
          <textarea
            maxLength={4000}
            onChange={(event) => onCommentBodyChange(event.target.value)}
            placeholder="Share context, decisions, or next steps"
            rows={3}
            value={commentBody}
          />
        </label>
        <button disabled={!canCreateComment} type="submit">
          {isCreatingComment ? "Posting..." : "Post comment"}
        </button>
      </form>
    </section>
  );
}
