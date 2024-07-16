import Button from 'hew/Button';
import Icon from 'hew/Icon';
import Message from 'hew/Message';
import Surface from 'hew/Surface';
import Tooltip from 'hew/Tooltip';
import Tree, { TreeDataNode } from 'hew/Tree';
import { isArray } from 'lodash';
import { useMemo } from 'react';

import { JsonObject, TrialDetails } from 'types';
import { downloadText } from 'utils/browser';
import { isJsonObject } from 'utils/data';

import css from './Metadata.module.scss';
import Section from './Section';

export interface Props {
  trial?: TrialDetails;
}

export const EMPTY_MESSAGE = 'No metadata found';

const getNodes = (data: JsonObject): TreeDataNode[] => {
  return Object.entries(data).map(([key, value]) => {
    if (isJsonObject(value)) {
      return {
        children: getNodes(value),
        key,
        selectable: false,
        title: <span className={css.node}>{key}</span>,
      };
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
        selectable: false,
        title: (
          <>
            <span className={css.key}>{key}:</span> <span className={css.node}>{stringValue}</span>
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
        <Tooltip content="Download metadata" key="download" placement="left">
          <Button
            disabled={!treeData.length}
            icon={<Icon decorative name="download" />}
            type="text"
            onClick={downloadMetadata}
          />
        </Tooltip>,
      ]}
      title="Metadata">
      <Surface>
        {treeData.length ? (
          <Tree defaultExpandAll treeData={treeData} />
        ) : (
          <Message title={EMPTY_MESSAGE} />
        )}
      </Surface>
    </Section>
  );
};

export default Metadata;
