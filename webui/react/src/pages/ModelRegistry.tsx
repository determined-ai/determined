import React, { useCallback, useState } from 'react';

import Page from 'components/Page';
import Section from 'components/Section';
import handleError, { ErrorType } from 'ErrorHandler';
import usePolling from 'hooks/usePolling';
import { getModels } from 'services/api';
import { ModelItem } from 'types';
import { isEqual } from 'utils/data';

const ModelRegistry: React.FC = () => {
  const [ models, setModels ] = useState<ModelItem[]>([]);

  const fetchModels = useCallback(async () => {
    try {
      const response = await getModels({});
      setModels(prev => {
        if (isEqual(prev, response.models)) return prev;
        return response.models;
      });
    } catch(e) {
      handleError({ message: 'Unable to fetch models.', silent: true, type: ErrorType.Api });
    }
  }, []);

  usePolling(fetchModels);

  return (
    <Page docTitle="Model Registry" id="models">
      <Section title="Model Registry">
        <div />
      </Section>
    </Page>);
};

export default ModelRegistry;
