import { render, screen } from '@testing-library/react';
import React from 'react';

import useFeature, { ValidFeature } from './useFeature';

const FeatureTest: React.FC = () => {
  const feature = useFeature();

  return (
    <ul>
      <li>{feature.isOn('trials_comparison' as ValidFeature) && 'trials_comparison'}</li>
      <li>{feature.isOn('unexist_feature' as ValidFeature) && 'unexist_feature'}</li>
    </ul>
  );
};

const setup = () => {
  return render(<FeatureTest />);
};

describe('useFeature', () => {
  // TODO: add test for a feature flag being on
  it('trials_comparison feature is not on', () => {
    setup();
    expect(screen.queryByText('trials_comparison')).not.toBeInTheDocument();
  });
  it('unexist_feature feature is not on', () => {
    setup();
    expect(screen.queryByText('unexist_feature')).not.toBeInTheDocument();
  });
});
