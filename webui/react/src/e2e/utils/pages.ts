import { isRbacEnabled } from 'e2e/utils/rbac';

export const defaultURL = isRbacEnabled() ? /workspaces/ : /dashboard/;

export const defaultTitle = isRbacEnabled() ? 'Workspaces' : 'Home';
