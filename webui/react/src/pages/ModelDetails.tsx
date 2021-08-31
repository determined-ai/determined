import { DownOutlined } from '@ant-design/icons';
import { Button, Card, Dropdown, Menu, Tooltip } from 'antd';
import React, { useCallback, useMemo, useState } from 'react';
import { useParams } from 'react-router-dom';

import Icon from 'components/Icon';
import InfoBox from 'components/InfoBox';
import Message, { MessageType } from 'components/Message';
import Page from 'components/Page';
import Section from 'components/Section';
import Spinner from 'components/Spinner';
import usePolling from 'hooks/usePolling';
import { getModel } from 'services/api';
import { isAborted, isNotFound } from 'services/utils';
import { ModelItem } from 'types';
import { isEqual } from 'utils/data';

import CollapsableCard from './ModelDetails/CollapsableCard';
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

  const metadata = useMemo(() => {
    return Object.entries(model?.metadata || {}).map((pair) => {
      return ({ content: pair[1], label: pair[0] });
    });
  }, [ model?. metadata ]);

  const referenceText = useMemo(() => {
    return (
      `from determined.experimental import Determined
model = Determined.getModel("${model?.name}")
ckpt = model.get_version("1234")
ckpt_path = ckpt.download()
ckpt = torch.load(os.path.join(ckpt_path, 'state_dict.pth'))

# WARNING: From here on out, this might not be possible to automate. Requires research.
from model import build_model
model = build_model()
model.load_state_dict(ckpt['models_state_dict'][0])

# If you get this far, you should be able to run \`model.eval()\``);
  }, [ model?.name ]);

  const handleCopy = useCallback(async () => {
    await navigator.clipboard.writeText(referenceText);
  }, [ referenceText ]);

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
        <div style={{
          display: 'flex',
          flexDirection: 'column',
          gap: 12,
          marginLeft: 20,
          marginRight: 20,
        }}>
          <CollapsableCard title={'Metadata'}>
            <InfoBox rows={metadata} />
            <Button type="link">add row</Button>
          </CollapsableCard>
          <CollapsableCard
            extra={(
              <Tooltip title="Copied!" trigger="click">
                <Button type="link" onClick={handleCopy}>Copy to clipboard</Button>
              </Tooltip>
            )}
            title={<>How to reference this model <Icon name="info" /></>}>
            <pre>{referenceText}</pre>
          </CollapsableCard>
          <Section
            divider
            title={detailsHeader}>
            <div style={{ display: 'flex', flexWrap: 'wrap', gap: 18 }}>
              <Card style={{ flexBasis: '66%' }} title="Source">
                <InfoBox rows={[]} />
              </Card>
              <div style={{ display: 'flex', flexDirection: 'column', flexGrow: 1, gap: 18 }}>
                <Card title="Validation Metrics">
                  <InfoBox rows={[]} />
                </Card>
                <Card title="Metadata">
                  <InfoBox rows={[]} />
                </Card>
              </div>
            </div>
          </Section>
        </div>
      </Page>
    </>
  );
};

export default ModelDetails;
