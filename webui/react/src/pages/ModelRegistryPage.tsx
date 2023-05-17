import React, { useRef } from 'react';

import Page from 'components/Page';

import ModelRegistry from '../components/ModelRegistry';

const ModelRegistryPage: React.FC = () => {
  const pageRef = useRef<HTMLElement>(null);

  return (
    <Page containerRef={pageRef} id="models" title="Model Registry">
      <ModelRegistry />
    </Page>
  );
};

export default ModelRegistryPage;
