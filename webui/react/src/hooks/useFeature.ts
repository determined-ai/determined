import queryString from 'query-string';
import { useLocation } from 'react-router-dom';

import { useStore } from 'contexts/Store';

type ValidFeature = 'rbac' // Add new feature switches here using `|`
interface FeatureHook {
  isOn: (feature: ValidFeature) => boolean
}

const useFeature = (): FeatureHook => {
  return { isOn: (ValidFeature) => IsOn(ValidFeature) };
};

const IsOn = (feature: string): boolean => {
  const location = useLocation();
  const queryParams = queryString.parse(location.search);
  const { info: { rbacEnabled } } = useStore();
  switch (feature) {
    case 'rbac':
      return rbacEnabled || queryParams[`f_${feature}`] === 'on';
    default:
      return queryParams[`f_${feature}`] === 'on';
  }
};

export default useFeature;
