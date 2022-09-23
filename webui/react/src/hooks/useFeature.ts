import queryString from 'query-string';

import { useStore } from 'contexts/Store';
import { DeterminedInfo } from 'types';

type ValidFeature = 'rbac' | 'mock_workspace_members'; // Add new feature switches here using `|`
const queryParams = queryString.parse(window.location.search);

interface FeatureHook {
  isOn: (feature: ValidFeature) => boolean;
}

const useFeature = (): FeatureHook => {
  const { info } = useStore();
  return { isOn: (ValidFeature) => IsOn(ValidFeature, info) };
};

const IsOn = (feature: string, info: DeterminedInfo): boolean => {
  const { rbacEnabled } = info;
  switch (feature) {
    case 'rbac':
      return rbacEnabled || queryParams[`f_${feature}`] === 'on';
    case 'mock_workspace_members':
        return rbacEnabled || queryParams[`f_${feature}`] === 'on';
    default:
      return queryParams[`f_${feature}`] === 'on';
  }
};

export default useFeature;
