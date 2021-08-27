import { DownOutlined } from '@ant-design/icons';
import { Button, Dropdown, Menu } from 'antd';
import React, { useCallback, useMemo, useState } from 'react';
import { useParams } from 'react-router-dom';

import Icon from 'components/Icon';
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

  const versionMenu = useMemo(() => {
    return <Menu>
      <Menu.Item key={0}>Version 0</Menu.Item>
    </Menu>;
  }, []);

  const detailsHeader = useMemo(() => {
    return <div style={{
      display: 'flex',
      justifyContent: 'space-between',
      width: '100%',
    }}
    ><p style={{ margin: 0 }}>Model Details</p>
      <div style={{ display: 'flex', gap: 4, height: '100%' }}>
        <Dropdown overlay={versionMenu}>
          <Button>
            Version 0 <DownOutlined />
          </Button>
        </Dropdown>
        <Button><Icon name="overflow-horizontal" size="tiny" /></Button></div></div>;
  }, [ versionMenu ]);

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
      <Page docTitle="Model Details" id="modelDetails">
        <Section
          divider
          title={detailsHeader}>
          {''}
        </Section>
      </Page>
    </>
  );
};

export default ModelDetails;
