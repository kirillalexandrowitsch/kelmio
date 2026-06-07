import { useState } from "react";
import type { CurrentUser, RuntimeVersion } from "../lib/api";
import type { AppSection } from "../lib/routing";

type SessionAccountControllerOptions = {
  initialSection: AppSection;
  initialSprintId: string;
  initialInviteAcceptToken: string | null;
};

export function useSessionAccountController({
  initialSection,
  initialSprintId,
  initialInviteAcceptToken,
}: SessionAccountControllerOptions) {
  const [user, setUser] = useState<CurrentUser | null>(null);
  const [loginValue, setLoginValue] = useState("admin");
  const [password, setPassword] = useState("admin12345");
  const [error, setError] = useState("");
  const [isBooting, setIsBooting] = useState(true);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [isLoggingOut, setIsLoggingOut] = useState(false);
  const [activeSection, setActiveSection] = useState<AppSection>(initialSection);
  const [routeSprintId, setRouteSprintId] = useState(initialSprintId);
  const [inviteAcceptToken, setInviteAcceptToken] = useState(
    initialInviteAcceptToken,
  );
  const [accountError, setAccountError] = useState("");
  const [accountSuccess, setAccountSuccess] = useState("");
  const [accountDisplayName, setAccountDisplayName] = useState("");
  const [runtimeVersion, setRuntimeVersion] = useState<RuntimeVersion | null>(
    null,
  );
  const [runtimeVersionError, setRuntimeVersionError] = useState("");
  const [isLoadingRuntimeVersion, setIsLoadingRuntimeVersion] = useState(false);
  const [isUpdatingProfile, setIsUpdatingProfile] = useState(false);
  const [currentPassword, setCurrentPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [confirmNewPassword, setConfirmNewPassword] = useState("");
  const [isChangingPassword, setIsChangingPassword] = useState(false);

  return {
    user,
    setUser,
    loginValue,
    setLoginValue,
    password,
    setPassword,
    error,
    setError,
    isBooting,
    setIsBooting,
    isSubmitting,
    setIsSubmitting,
    isLoggingOut,
    setIsLoggingOut,
    activeSection,
    setActiveSection,
    routeSprintId,
    setRouteSprintId,
    inviteAcceptToken,
    setInviteAcceptToken,
    accountError,
    setAccountError,
    accountSuccess,
    setAccountSuccess,
    accountDisplayName,
    setAccountDisplayName,
    runtimeVersion,
    setRuntimeVersion,
    runtimeVersionError,
    setRuntimeVersionError,
    isLoadingRuntimeVersion,
    setIsLoadingRuntimeVersion,
    isUpdatingProfile,
    setIsUpdatingProfile,
    currentPassword,
    setCurrentPassword,
    newPassword,
    setNewPassword,
    confirmNewPassword,
    setConfirmNewPassword,
    isChangingPassword,
    setIsChangingPassword,
  };
}
