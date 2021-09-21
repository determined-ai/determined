import { Button, Card, Tabs, Tooltip } from 'antd';
import React, { useCallback, useMemo, useState } from 'react';
import { useParams } from 'react-router';

import Icon from 'components/Icon';
import InfoBox, { InfoRow } from 'components/InfoBox';
import Message, { MessageType } from 'components/Message';
import Page from 'components/Page';
import Spinner from 'components/Spinner';
import usePolling from 'hooks/usePolling';
import { getModelVersion } from 'services/api';
import { isAborted, isNotFound } from 'services/utils';
import { ModelVersion } from 'types';
import { isEqual } from 'utils/data';

import ModelVersionHeader from './ModelVersionDetails/ModelVersionHeader';

const { TabPane } = Tabs;

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

  const referenceText = useMemo(() => {
    return (
      `from determined.experimental import Determined
model = Determined.getModel("${modelVersion?.model?.name}")
ckpt = model.get_version("${modelVersion?.version}")
ckpt_path = ckpt.download()
ckpt = torch.load(os.path.join(ckpt_path, 'state_dict.pth'))

# WARNING: From here on out, this might not be possible to automate. Requires research.
from model import build_model
model = build_model()
model.load_state_dict(ckpt['models_state_dict'][0])

# If you get this far, you should be able to run \`model.eval()\``);
  }, [ modelVersion ]);

  const handleCopy = useCallback(async () => {
    await navigator.clipboard.writeText(referenceText);
  }, [ referenceText ]);

  /*
  const metadata = useMemo(() => {
    return Object.entries(model?.model.metadata || {}).map((pair) => {
      return ({ content: pair[1], label: pair[0] });
    });
  }, [ modelVersion.metadata ]);
  */

  const metadata = Object.entries({}).map((pair) => {
    return ({ content: pair[1], label: pair[0] } as InfoRow);
  });

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
      bodyNoPadding
      docTitle="Model Version Details"
      headerComponent={<ModelVersionHeader modelVersion={modelVersion} />}
      id="modelDetails">
      <Tabs
        defaultActiveKey="1"
        tabBarStyle={{ backgroundColor: 'var(--theme-colors-monochrome-17)', paddingLeft: 36 }}>
        <TabPane key="1" tab="Overview">
          <div style={{
            display: 'flex',
            flexDirection: 'column',
            gap: 12,
            padding: 36,
          }}>
            {metadata.length > 0 &&
        <Card title={'Metadata'}>
          <InfoBox rows={metadata} />
          <Button type="link">add row</Button>
        </Card>
            }
            <Card
              extra={(
                <Tooltip title="Copied!" trigger="click">
                  <Button type="link" onClick={handleCopy}>Copy to clipboard</Button>
                </Tooltip>
              )}
              title={<>How to reference this model <Icon name="info" /></>}>
              <pre>{referenceText}</pre>
            </Card>
          </div>
        </TabPane>
        <TabPane key="2" tab="Checkpoint Details">
          <div style={{
            display: 'flex',
            flexDirection: 'column',
            gap: 12,
            marginLeft: 20,
            marginRight: 20,
          }} />
        </TabPane>
      </Tabs>
    </Page>
  );
};

export default ModelVersionDetails;
