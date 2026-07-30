import { DashboardLayout } from "@/features/auth/dashboard-layout";
import { ProfilePage } from "@/features/auth/profile-page";

export default function ProfileRoute() {
  return <DashboardLayout><ProfilePage /></DashboardLayout>;
}
