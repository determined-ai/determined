import queryString from 'query-string';

import { useStore } from 'contexts/Store';

type ValidFeature = 'rbac' // Add new feature switches here using `|`
const queryParams = queryString.parse(window.location.search);

interface FeatureHook {
  isOn: (feature: ValidFeature) => boolean
}

const useFeature = (): FeatureHook => {
  return { isOn: (ValidFeature) => IsOn(ValidFeature) };
};

const IsOn = (feature: string): boolean => {
  const { info: { rbacEnabled } } = useStore();
  switch (feature) {
    case 'rbac':
      return rbacEnabled || queryParams[`f_${feature}`] === 'on';
    default:
      return queryParams[`f_${feature}`] === 'on';
  }
};

export default useFeature;
