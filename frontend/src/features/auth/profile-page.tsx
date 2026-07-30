"use client";

import { ChangeEvent, FormEvent, useEffect, useState } from "react";
import { useRouter } from "next/navigation";

import { AuthAPIError, avatarURL } from "@/features/auth/api";
import { useAuth } from "@/features/auth/auth-provider";

export function ProfilePage() {
  const { status, user } = useAuth();
  const router = useRouter();

  if (status === "loading") return <main className="flex min-h-screen items-center justify-center bg-zinc-50 text-sm text-zinc-600">Восстанавливаем сессию…</main>;
  if (status !== "authenticated" || !user) {
    router.replace("/login");
    return null;
  }

  return (
    <main className="mx-auto w-full max-w-6xl px-6 py-10">
      <div className="max-w-xl">
        <button className="text-sm font-medium text-blue-700 hover:text-blue-800" type="button" onClick={() => router.back()}>← Назад</button>
        <h1 className="mt-5 text-3xl font-semibold tracking-tight">Профиль</h1>
        <p className="mt-2 text-zinc-600">Просмотрите данные аккаунта и измените отображаемое имя.</p>
        <ProfileForm user={user} />
      </div>
    </main>
  );
}

function ProfileForm({ user }: { user: NonNullable<ReturnType<typeof useAuth>["user"]> }) {
  const { updateProfile, uploadAvatar } = useAuth();
  const [name, setName] = useState(user.name);
  const [avatarFile, setAvatarFile] = useState<File | null>(null);
  const [avatarPreview, setAvatarPreview] = useState(avatarURL(user.avatar_url));
  const [isUploading, setIsUploading] = useState(false);
  const [error, setError] = useState("");
  const [isSaving, setIsSaving] = useState(false);
  const [saved, setSaved] = useState(false);

  useEffect(() => {
    return () => {
      if (avatarPreview?.startsWith("blob:")) URL.revokeObjectURL(avatarPreview);
    };
  }, [avatarPreview]);

  function handleAvatarChange(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0] ?? null;
    setError("");
    setSaved(false);
    if (file && !["image/jpeg", "image/png"].includes(file.type)) {
      setAvatarFile(null);
      event.target.value = "";
      setError("Выберите изображение в формате JPEG или PNG.");
      return;
    }
    if (file && file.size > 5 * 1024 * 1024) {
      setAvatarFile(null);
      event.target.value = "";
      setError("Размер аватара не должен превышать 5 МБ.");
      return;
    }
    setAvatarFile(file);
    setAvatarPreview(file ? URL.createObjectURL(file) : avatarURL(user.avatar_url));
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");
    setSaved(false);
    setIsSaving(true);
    try {
      if (avatarFile) {
        setIsUploading(true);
        const updatedUser = await uploadAvatar(avatarFile);
        setAvatarFile(null);
        setAvatarPreview(avatarURL(updatedUser.avatar_url));
      }
      await updateProfile(name);
      setSaved(true);
    } catch (caught) {
      setError(caught instanceof AuthAPIError ? caught.message : "Не удалось сохранить профиль.");
    } finally {
      setIsSaving(false);
      setIsUploading(false);
    }
  }

  return (
    <form className="mt-8 rounded-2xl border border-zinc-200 bg-white p-6 shadow-sm" onSubmit={handleSubmit}>
      <div>
        <p className="text-sm font-medium">Аватар</p>
        <div className="mt-3 flex items-center gap-4">
          {avatarPreview ? <img className="size-20 rounded-full object-cover" src={avatarPreview} alt="Предпросмотр аватара" /> : <span className="flex size-20 items-center justify-center rounded-full bg-blue-100 text-2xl font-semibold text-blue-700">{user.name.slice(0, 1).toUpperCase()}</span>}
          <label className="cursor-pointer rounded-lg border border-zinc-300 px-4 py-2.5 text-sm font-medium hover:bg-zinc-50">
            Загрузить аватар
            <input className="sr-only" type="file" accept="image/jpeg,image/png" onChange={handleAvatarChange} />
          </label>
        </div>
        <p className="mt-2 text-xs text-zinc-500">JPEG или PNG, до 5 МБ.</p>
      </div>
      <label className="block text-sm font-medium" htmlFor="profile-email">Email</label>
      <input className="mt-2 w-full rounded-lg border border-zinc-200 bg-zinc-100 px-3 py-2.5 text-zinc-500" id="profile-email" value={user.email} readOnly />
      <label className="mt-5 block text-sm font-medium" htmlFor="profile-name">Имя</label>
      <input className="mt-2 w-full rounded-lg border border-zinc-300 px-3 py-2.5 outline-none focus:border-blue-600 focus:ring-2 focus:ring-blue-100" id="profile-name" maxLength={100} required value={name} onChange={(event) => setName(event.target.value)} />
      {error ? <p className="mt-3 text-sm text-red-700" role="alert">{error}</p> : null}
      {saved ? <p className="mt-3 text-sm text-green-700" role="status">Профиль сохранён.</p> : null}
      <div className="mt-6 flex justify-end">
        <button className="rounded-lg bg-blue-700 px-4 py-2.5 font-medium text-white hover:bg-blue-800 disabled:cursor-not-allowed disabled:opacity-60" type="submit" disabled={isSaving || isUploading}>{isSaving || isUploading ? "Сохраняем…" : "Сохранить"}</button>
      </div>
    </form>
  );
}
