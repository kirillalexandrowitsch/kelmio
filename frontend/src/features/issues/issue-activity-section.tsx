import { FormError } from "../../components/form-feedback";
import { type IssueActivity, type TeamMember } from "../../lib/api-types";
import { activityDescription, activityTitle } from "../../lib/activity-view";
import { formatDateTime } from "../../lib/formatting";

type IssueActivitySectionProps = {
  activity: IssueActivity[];
  activityError: string;
  isLoadingActivity: boolean;
  teamMembers: TeamMember[];
};

export function IssueActivitySection({
  activity,
  activityError,
  isLoadingActivity,
  teamMembers,
}: IssueActivitySectionProps) {
  return (
    <section className="activity-section" aria-label="Issue activity">
      <header className="comments-header">
        <div>
          <p className="eyebrow">Activity</p>
          <h3>{activity.length}</h3>
        </div>
        {isLoadingActivity ? (
          <span className="muted">Loading activity</span>
        ) : null}
      </header>

      {activityError ? <FormError message={activityError} /> : null}

      {activity.length > 0 ? (
        <div className="activity-list">
          {activity.map((activityItem) => {
            const description = activityDescription(activityItem, teamMembers);

            return (
              <article className="activity-card" key={activityItem.id}>
                <span className="activity-dot" aria-hidden="true" />
                <div>
                  <header>
                    <strong>{activityTitle(activityItem)}</strong>
                    <span>{formatDateTime(activityItem.created_at)}</span>
                  </header>
                  <p>
                    {activityItem.actor_display_name ?? "System"}
                    {description ? ` · ${description}` : ""}
                  </p>
                </div>
              </article>
            );
          })}
        </div>
      ) : (
        <div className="comments-empty">No activity yet</div>
      )}
    </section>
  );
}
