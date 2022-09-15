import queryString from 'query-string';
import { useLocation } from 'react-router-dom';

import { useStore } from 'contexts/Store';

interface FeatureHook {
  isOnRbac: () => boolean
}

interface Queries {
  rbac?: string;
}

const useFeature = (): FeatureHook => {
  return { isOnRbac: () => IsOnRbac() };
};

const IsOnRbac = (): boolean => {
  const { info: { rbacEnabled } } = useStore();
  const location = useLocation();
  const { rbac }: Queries = queryString.parse(location.search);
  return rbacEnabled || rbac === 'on';
};

export default useFeature;
