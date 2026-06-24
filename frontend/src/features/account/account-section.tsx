import { type FormEvent } from "react";

import { FormError } from "../../components/form-feedback";
import { type CurrentUser, type RuntimeVersion } from "../../lib/api-types";
import { runtimeVersionDisplay } from "../../lib/runtime-version";
import {
  AtSign,
  BadgeCheck,
  Fingerprint,
  KeyRound,
  Mail,
  Save,
  ServerCog,
  UserRound,
} from "lucide-react";
import { Badge, Button, Field, Input } from "../../ui";

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
          <UserRound size={17} />
          <span>Display name</span>
          <strong>{user.display_name}</strong>
        </div>
        <div>
          <AtSign size={17} />
          <span>Username</span>
          <strong>@{user.username}</strong>
        </div>
        <div>
          <Mail size={17} />
          <span>Email</span>
          <strong>{user.email}</strong>
        </div>
        <div>
          <BadgeCheck size={17} />
          <span>Role</span>
          <Badge tone="success">{user.workspace.role}</Badge>
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
              <ServerCog size={17} />
              <span>Version</span>
              <strong>{versionDisplay.version}</strong>
            </div>
            <div>
              <Fingerprint size={17} />
              <span>Commit</span>
              <strong>{versionDisplay.commit}</strong>
            </div>
            <div>
              <ServerCog size={17} />
              <span>Environment</span>
              <strong>{versionDisplay.environment}</strong>
            </div>
            <div>
              <ServerCog size={17} />
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

        <Field label="Display name">
          <Input
            maxLength={80}
            onChange={(event) => onDisplayNameChange(event.target.value)}
            value={accountDisplayName}
          />
        </Field>

        <Button disabled={!canUpdateProfile} icon={<Save size={16} />} type="submit">
          {isUpdatingProfile ? "Saving..." : "Save profile"}
        </Button>
      </form>

      <form className="account-form" onSubmit={onChangePassword}>
        <header className="section-header">
          <div>
            <p className="eyebrow">Security</p>
            <h2>Change password</h2>
          </div>
        </header>

        <Field label="Current password">
          <Input
            autoComplete="current-password"
            onChange={(event) => onCurrentPasswordChange(event.target.value)}
            type="password"
            value={currentPassword}
          />
        </Field>
        <Field label="New password">
          <Input
            autoComplete="new-password"
            minLength={8}
            onChange={(event) => onNewPasswordChange(event.target.value)}
            type="password"
            value={newPassword}
          />
        </Field>
        <Field label="Confirm new password">
          <Input
            autoComplete="new-password"
            minLength={8}
            onChange={(event) => onConfirmNewPasswordChange(event.target.value)}
            type="password"
            value={confirmNewPassword}
          />
        </Field>

        <Button disabled={!canChangePassword} icon={<KeyRound size={16} />} type="submit">
          {isChangingPassword ? "Changing..." : "Change password"}
        </Button>
      </form>
    </section>
  );
}
