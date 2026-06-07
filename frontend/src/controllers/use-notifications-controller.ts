import { useState } from "react";
import type { AppNotification } from "../lib/api";

export function useNotificationsController() {
  const [notifications, setNotifications] = useState<AppNotification[]>([]);
  const [notificationsError, setNotificationsError] = useState("");
  const [isLoadingNotifications, setIsLoadingNotifications] = useState(false);
  const [unreadNotificationsCount, setUnreadNotificationsCount] = useState(0);
  const [isNotificationsOpen, setIsNotificationsOpen] = useState(false);

  return {
    notifications,
    setNotifications,
    notificationsError,
    setNotificationsError,
    isLoadingNotifications,
    setIsLoadingNotifications,
    unreadNotificationsCount,
    setUnreadNotificationsCount,
    isNotificationsOpen,
    setIsNotificationsOpen,
  };
}
