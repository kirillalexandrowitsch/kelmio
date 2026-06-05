import { type FormEvent } from "react";

import { FormError } from "../../components/form-feedback";
import { type CurrentUser, type RuntimeVersion } from "../../lib/api-types";
import { runtimeVersionDisplay } from "../../lib/runtime-version";

type AccountSectionProps = {
  accountDisplayName: string;
  accountError: string;
  accountSuccess: string;
  canChangePassword: boolean;
  canUpdateProfile: boolean;
  confirmNewPassword: string;
  currentPassword: string;
  isActive: boolean;
  isChangingPassword: boolean;
  isLoadingRuntimeVersion: boolean;
  isUpdatingProfile: boolean;
  newPassword: string;
  onChangePassword: (event: FormEvent<HTMLFormElement>) => void;
  onConfirmNewPasswordChange: (value: string) => void;
  onCurrentPasswordChange: (value: string) => void;
  onDisplayNameChange: (value: string) => void;
  onNewPasswordChange: (value: string) => void;
  onUpdateProfile: (event: FormEvent<HTMLFormElement>) => void;
  runtimeVersion: RuntimeVersion | null;
  runtimeVersionError: string;
  user: CurrentUser;
};

export function AccountSection({
  accountDisplayName,
  accountError,
  accountSuccess,
  canChangePassword,
  canUpdateProfile,
  confirmNewPassword,
  currentPassword,
  isActive,
  isChangingPassword,
  isLoadingRuntimeVersion,
  isUpdatingProfile,
  newPassword,
  onChangePassword,
  onConfirmNewPasswordChange,
  onCurrentPasswordChange,
  onDisplayNameChange,
  onNewPasswordChange,
  onUpdateProfile,
  runtimeVersion,
  runtimeVersionError,
  user,
}: AccountSectionProps) {
  const versionDisplay = runtimeVersionDisplay(runtimeVersion);

  return (
    <section
      className="account-panel"
      aria-label="Account settings"
      hidden={!isActive}
    >
      <header className="section-header">
        <div>
          <p className="eyebrow">Account</p>
          <h2>Profile and password</h2>
        </div>
      </header>

      <div className="account-card">
        <div>
          <span>Display name</span>
          <strong>{user.display_name}</strong>
        </div>
        <div>
          <span>Username</span>
          <strong>@{user.username}</strong>
        </div>
        <div>
          <span>Email</span>
          <strong>{user.email}</strong>
        </div>
        <div>
          <span>Role</span>
          <strong>{user.workspace.role}</strong>
        </div>
      </div>

      <section className="account-form account-metadata-panel">
        <header className="section-header">
          <div>
            <p className="eyebrow">Deployment</p>
            <h2>Runtime metadata</h2>
          </div>
        </header>

        {isLoadingRuntimeVersion ? (
          <p className="muted">Loading deployment metadata...</p>
        ) : null}
        <FormError message={runtimeVersionError} />

        {runtimeVersion ? (
          <div className="account-card account-metadata-card">
            <div>
              <span>Version</span>
              <strong>{versionDisplay.version}</strong>
            </div>
            <div>
              <span>Commit</span>
              <strong>{versionDisplay.commit}</strong>
            </div>
            <div>
              <span>Environment</span>
              <strong>{versionDisplay.environment}</strong>
            </div>
            <div>
              <span>Build time</span>
              <strong>{versionDisplay.buildTime}</strong>
            </div>
          </div>
        ) : !isLoadingRuntimeVersion && !runtimeVersionError ? (
          <p className="muted">Deployment metadata is not available.</p>
        ) : null}
      </section>

      <FormError message={accountError} />
      {accountSuccess ? <p className="form-success">{accountSuccess}</p> : null}

      <form className="account-form" onSubmit={onUpdateProfile}>
        <header className="section-header">
          <div>
            <p className="eyebrow">Profile</p>
            <h2>Display name</h2>
          </div>
        </header>

        <label>
          <span>Display name</span>
          <input
            maxLength={80}
            onChange={(event) => onDisplayNameChange(event.target.value)}
            value={accountDisplayName}
          />
        </label>

        <button disabled={!canUpdateProfile} type="submit">
          {isUpdatingProfile ? "Saving..." : "Save profile"}
        </button>
      </form>

      <form className="account-form" onSubmit={onChangePassword}>
        <header className="section-header">
          <div>
            <p className="eyebrow">Security</p>
            <h2>Change password</h2>
          </div>
        </header>

        <label>
          <span>Current password</span>
          <input
            autoComplete="current-password"
            onChange={(event) => onCurrentPasswordChange(event.target.value)}
            type="password"
            value={currentPassword}
          />
        </label>
        <label>
          <span>New password</span>
          <input
            autoComplete="new-password"
            minLength={8}
            onChange={(event) => onNewPasswordChange(event.target.value)}
            type="password"
            value={newPassword}
          />
        </label>
        <label>
          <span>Confirm new password</span>
          <input
            autoComplete="new-password"
            minLength={8}
            onChange={(event) => onConfirmNewPasswordChange(event.target.value)}
            type="password"
            value={confirmNewPassword}
          />
        </label>

        <button disabled={!canChangePassword} type="submit">
          {isChangingPassword ? "Changing..." : "Change password"}
        </button>
      </form>
    </section>
  );
}
