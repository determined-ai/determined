import { render, screen } from '@testing-library/react';
import React from 'react';
import { BrowserRouter } from 'react-router-dom';

import useFeature, { ValidFeature } from './useFeature';

const FeatureTest: React.FC = () => {
  const feature = useFeature();

  return (
    <ul>
      <li>{feature.isOn('unexist_feature' as ValidFeature) && 'unexist_feature'}</li>
    </ul>
  );
};

const setup = () => {
  return render(
    <BrowserRouter>
      <FeatureTest />
    </BrowserRouter>,
  );
};

describe('useFeature', () => {
  // TODO: add test for a feature flag being on
  it('unexist_feature feature is not on', () => {
    setup();
    expect(screen.queryByText('unexist_feature')).not.toBeInTheDocument();
  });
});
