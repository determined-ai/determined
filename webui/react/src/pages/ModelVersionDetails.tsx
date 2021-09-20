import React, { useCallback, useState } from 'react';
import { useParams } from 'react-router';

import usePolling from 'hooks/usePolling';
import { getModelVersion } from 'services/api';
import { isAborted } from 'services/utils';
import { ModelVersion } from 'types';
import { isEqual } from 'utils/data';

interface Params {
  modelId: string;
  versionId: string;
}

const ModelVersionDetails: React.FC = () => {
  const [ modelVersion, setModelVersion ] = useState<ModelVersion>();
  const { modelId, versionId } = useParams<Params>();
  const [ pageError, setPageError ] = useState<Error>();

  const fetchModelVersion = useCallback(async () => {
    try {
      const versionData = await getModelVersion(
        { modelName: 'mnist-prod', versionId: 2 },
      );
      if (!isEqual(versionData, modelVersion)) setModelVersion(versionData);
    } catch (e) {
      if (!pageError && !isAborted(e)) setPageError(e as Error);
    }
  }, [ modelVersion, pageError ]);

  usePolling(fetchModelVersion);

  return (
    <div />
  );
};

export default ModelVersionDetails;
