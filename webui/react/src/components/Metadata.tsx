import Button from 'hew/Button';
import Icon from 'hew/Icon';
import Message from 'hew/Message';
import Tree, { TreeDataNode } from 'hew/Tree';
import { useCallback, useMemo } from 'react';

import { JsonObject, TrialDetails } from 'types';
import { downloadText } from 'utils/browser';
import { isJsonObject } from 'utils/data';

import Section from './Section';

export interface Props {
  trial?: TrialDetails;
}

export const EMPTY_MESSAGE = 'No metadata found';

const Metadata: React.FC<Props> = ({ trial }: Props) => {
  const getNodes = useCallback((data: JsonObject): TreeDataNode[] => {
    return Object.entries(data).map(([key, value]) => {
      if (isJsonObject(value)) {
        return { children: getNodes(value), key, title: key };
      } else if (value) {
        const stringValue = value.toString();
        return { children: [{ key: stringValue, title: stringValue }], key, title: key };
      }
      return { children: [{ key: 'undefined', title: 'undefined' }], key, title: key };
    });
  }, []);

  const treeData: TreeDataNode[] = useMemo(() => {
    if (!trial?.metadata) return [];
    return getNodes(trial?.metadata);
  }, [trial?.metadata, getNodes]);

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
