import Button from 'hew/Button';
import Icon from 'hew/Icon';
import Surface from 'hew/Surface';
import { useTheme } from 'hew/Theme';
import Tooltip from 'hew/Tooltip';
import Tree, { TreeDataNode } from 'hew/Tree';
import { isArray } from 'lodash';

import { RawJson, TrialDetails } from 'types';
import { downloadText } from 'utils/browser';
import { isJsonObject } from 'utils/data';

import css from './Metadata.module.scss';
import Section from './Section';

interface Props {
  trial?: TrialDetails;
}

export const EMPTY_MESSAGE = 'No metadata found';

const Metadata: React.FC<Props> = ({ trial }: Props) => {
  const { tokens } = useTheme();

  const getNodes = (data: RawJson): TreeDataNode[] => {
    return Object.entries(data).map(([key, value]) => {
      if (isJsonObject(value) || isArray(value)) {
        return {
          children: getNodes(value),
          key,
          selectable: false,
          title: <span style={{ color: tokens.colorTextDescription }}>{key}</span>,
        };
      }
      return {
        key,
        selectable: false,
        title: (
          <>
            <span style={{ color: tokens.colorTextDescription }}>{key}:</span> <span>{value}</span>
          </>
        ),
      };
    });
  };

  const downloadMetadata = () => {
    downloadText(`${trial?.id}_metadata.json`, [JSON.stringify(trial?.metadata)]);
  };

  const treeData = (trial?.metadata && getNodes(trial?.metadata)) ?? [];

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
        <div className={css.base}>
          <Tree defaultExpandAll treeData={treeData} />
        </div>
      </Surface>
    </Section>
  );
};

export default Metadata;
