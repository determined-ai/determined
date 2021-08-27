import React, { useCallback, useState } from 'react';
import { useParams } from 'react-router-dom';

import Message, { MessageType } from 'components/Message';
import Page from 'components/Page';
import Section from 'components/Section';
import Spinner from 'components/Spinner';
import usePolling from 'hooks/usePolling';
import { getModel } from 'services/api';
import { isAborted, isNotFound } from 'services/utils';
import { ModelItem } from 'types';
import { isEqual } from 'utils/data';

import ModelHeader from './ModelDetails/ModelHeader';

interface Params {
  modelId: string;
}

const ModelDetails: React.FC = () => {
  const [ model, setModel ] = useState<ModelItem>();
  const { modelId } = useParams<Params>();
  const [ pageError, setPageError ] = useState<Error>();

  const id = parseInt(modelId);

  const fetchModel = useCallback(async () => {
    try {
      const modelData = await getModel({ modelName: 'mnist' });
      if (!isEqual(modelData, model)) setModel(modelData);
    } catch (e) {
      if (!pageError && !isAborted(e)) setPageError(e);
    }
  }, [ model, pageError ]);

  usePolling(fetchModel);

  if (isNaN(id)) {
    return <Message title={`Invalid Model ID ${modelId}`} />;
  } else if (pageError) {
    const message = isNotFound(pageError) ?
      `Unable to find model ${modelId}` :
      `Unable to fetch model ${modelId}`;
    return <Message title={message} type={MessageType.Warning} />;
  } else if (!model) {
    return <Spinner tip={`Loading model ${modelId} details...`} />;
  }

  return (
    <>
      <ModelHeader model={model} />
      <Page docTitle="Model Details" id="modelDetails" />
    </>
  );
};

export default ModelDetails;
