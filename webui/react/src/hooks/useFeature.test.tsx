import { render, screen } from '@testing-library/react';
import React, { useEffect, useMemo } from 'react';

import StoreProvider, { initInfo, StoreAction, useStoreDispatch } from 'contexts/Store';

import useFeature from './useFeature';

const FeatureTest: React.FC = () => {
  const storeDispatch = useStoreDispatch();
  const feature = useFeature();
  const testInfo = useMemo(() => ({ ...initInfo, featureSwitch: ['webhooks'] }), []);
  useEffect(() => {
    storeDispatch({ type: StoreAction.SetInfo, value: testInfo });
  }, [storeDispatch, testInfo]);

  return (
    <ul>
      <li>{feature.isOn('trials_comparison') && 'trials_comparison'}</li>
      <li>{feature.isOn('webhooks') && 'webhooks'}</li>
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
  it('trials_comparison feature is not on', async () => {
    setup();
    expect(screen.queryByText('trials_comparison')).not.toBeInTheDocument();
  });
  it('webhooks feature is on', async () => {
    setup();
    expect(screen.queryByText('webhooks')).toBeInTheDocument();
  });
});
