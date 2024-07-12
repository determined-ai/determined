import Button from 'hew/Button';
import Icon from 'hew/Icon';
import Message from 'hew/Message';
import Tree, { TreeDataNode } from 'hew/Tree';
import { isArray } from 'lodash';
import { useMemo } from 'react';

import { JsonObject, TrialDetails } from 'types';
import { downloadText } from 'utils/browser';
import { isJsonObject } from 'utils/data';

import Section from './Section';

export interface Props {
  trial?: TrialDetails;
}

export const EMPTY_MESSAGE = 'No metadata found';

const getNodes = (data: JsonObject): TreeDataNode[] => {
  return Object.entries(data).map(([key, value]) => {
    if (isJsonObject(value)) {
      return { children: getNodes(value), key, title: <strong>{key}</strong> };
    } else {
      let stringValue = '';
      if (value === null || value === undefined) {
        stringValue = 'undefined';
      } else if (isArray(value)) {
        stringValue = `[${value.join(', ')}]`;
      } else {
        stringValue = value.toString();
      }
      return {
        key,
        title: (
          <>
            <strong>{key}:</strong> {stringValue}
          </>
        ),
      };
    }
  });
};

const Metadata: React.FC<Props> = ({ trial }: Props) => {
  const treeData: TreeDataNode[] = useMemo(() => {
    if (!trial?.metadata) return [];
    return getNodes(trial?.metadata);
  }, [trial?.metadata]);

  const downloadMetadata = () => {
    if (trial?.metadata)
      downloadText(`${trial?.id}_metadata.json`, [JSON.stringify(trial?.metadata)]);
  };

  return (
    <Section
      options={[
        <Button
          disabled={!treeData.length}
          icon={<Icon decorative name="download" />}
          key="download"
          onClick={downloadMetadata}>
          Download
        </Button>,
      ]}
      title="Metadata">
      {treeData.length ? (
        <Tree defaultExpandAll treeData={treeData} />
      ) : (
        <Message title={EMPTY_MESSAGE} />
      )}
    </Section>
  );
};

export default Metadata;
