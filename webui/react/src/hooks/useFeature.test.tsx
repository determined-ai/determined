import { render, screen } from '@testing-library/react';
import React, { useEffect, useMemo } from 'react';

import StoreProvider, { initInfo, StoreAction, useStoreDispatch } from 'contexts/Store';

import useFeature, { ValidFeature } from './useFeature';

const FeatureTest: React.FC = () => {
  const storeDispatch = useStoreDispatch();
  const feature = useFeature();
  const testInfo = useMemo(() => ({ ...initInfo, featureSwitches: ['webhooks'] }), []);
  useEffect(() => {
    storeDispatch({ type: StoreAction.SetInfo, value: testInfo });
  }, [storeDispatch, testInfo]);

  return (
    <ul>
      <li>{feature.isOn('trials_comparison' as ValidFeature) && 'trials_comparison'}</li>
      <li>{feature.isOn('unexist_feature' as ValidFeature) && 'unexist_feature'}</li>
    </ul>
  );
};

const setup = () => {
  return render(
    <StoreProvider>
      <FeatureTest />
    </StoreProvider>,
  );
};

describe('useFeature', () => {
  it('trials_comparison feature is not on', () => {
    setup();
    expect(screen.queryByText('trials_comparison')).not.toBeInTheDocument();
  });
  it('unexist_feature feature is not on', () => {
    setup();
    expect(screen.queryByText('unexist_feature')).not.toBeInTheDocument();
  });
});
