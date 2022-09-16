import queryString from 'query-string';
import { useEffect, useState } from 'react';
import { useLocation } from 'react-router-dom';

import { useStore } from 'contexts/Store';

interface FeatureHook {
  isOn: (ValidFeature: string) => boolean
}

const useFeature = (): FeatureHook => {
  const location = useLocation();
  const [ features, setFeatures ] = useState<string[]>([]);
  useEffect(() => {
    const queries = queryString.parse(location.search);
    const parsedFeature: string[] = [];
    Object.keys(queries).forEach((k: string) => {
      k.startsWith('f_') && queries[k] === 'on' && parsedFeature.push(k.substring(2));
    });
    setFeatures(parsedFeature);
  }, []);

  return { isOn: (ValidFeature) => IsOn(ValidFeature, features) };
};

const IsOn = (ValidFeature: string, features: string[]): boolean => {
  const { info: { rbacEnabled } } = useStore();

  return rbacEnabled || features.includes(ValidFeature);
};

export default useFeature;
