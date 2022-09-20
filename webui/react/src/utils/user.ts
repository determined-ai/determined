import { DetailedUser, Group, Member, MemberOrGroup, User } from 'types';

interface UserNameFields {
  displayName?: string;
  username?: string;
}

export function getDisplayName(user: DetailedUser | User | UserNameFields | undefined): string {
  return user?.displayName || user?.username || 'Unavailable';
}

export function isMember(obj: MemberOrGroup): string | undefined {
  const member = obj as Member;
  return member?.username || member?.displayName;
};

export function getName(obj: MemberOrGroup): string {
  const member = obj as Member;
  const group = obj as Group;
  return isMember(obj) ? getDisplayName(member) : group.name;
};
