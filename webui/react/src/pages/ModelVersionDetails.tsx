import React, { useCallback, useState } from 'react';
import { useParams } from 'react-router';

import Message, { MessageType } from 'components/Message';
import Page from 'components/Page';
import Spinner from 'components/Spinner';
import usePolling from 'hooks/usePolling';
import { getModelVersion } from 'services/api';
import { isAborted, isNotFound } from 'services/utils';
import { ModelVersion } from 'types';
import { isEqual } from 'utils/data';

import ModelVersionHeader from './ModelVersionDetails/ModelVersionHeader';

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

  if (isNaN(parseInt(modelId))) {
    return <Message title={`Invalid Model ID ${modelId}`} />;
  } else if (isNaN(parseInt(versionId))) {
    return <Message title={`Invalid Version ID ${versionId}`} />;
  } else if (pageError) {
    const message = isNotFound(pageError) ?
      `Unable to find model ${modelId} version ${versionId}` :
      `Unable to fetch model ${modelId} version ${versionId}`;
    return <Message title={message} type={MessageType.Warning} />;
  } else if (!modelVersion) {
    return <Spinner tip={`Loading model ${modelId} version ${versionId} details...`} />;
  }

  return (
    <Page
      docTitle="Model Version Details"
      headerComponent={<ModelVersionHeader modelVersion={modelVersion} />}
      id="modelDetails">
      <div style={{
        display: 'flex',
        flexDirection: 'column',
        gap: 12,
        marginLeft: 20,
        marginRight: 20,
      }} /></Page>
  );
};

export default ModelVersionDetails;
