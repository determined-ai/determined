import { render, screen } from '@testing-library/react';
import React, { useEffect, useMemo } from 'react';

import { DeterminedInfoProvider, initInfo, useUpdateDeterminedInfo } from 'stores/determinedInfo';

import useFeature, { ValidFeature } from './useFeature';

const FeatureTest: React.FC = () => {
  const feature = useFeature();
  const testInfo = useMemo(() => ({ ...initInfo, featureSwitches: ['webhooks'] }), []);
  const updateInfo = useUpdateDeterminedInfo();
  useEffect(() => {
    updateInfo(testInfo);
  }, [testInfo, updateInfo]);

  return (
    <ul>
      <li>{feature.isOn('trials_comparison' as ValidFeature) && 'trials_comparison'}</li>
      <li>{feature.isOn('unexist_feature' as ValidFeature) && 'unexist_feature'}</li>
    </ul>
  );
};

const setup = () => {
  return render(
    <DeterminedInfoProvider>
      <FeatureTest />
    </DeterminedInfoProvider>,
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
