import { LeftOutlined } from '@ant-design/icons';
import { Breadcrumb, Button } from 'antd';
import React, { useMemo } from 'react';

import Icon from 'components/Icon';
import InfoBox, { InfoRow } from 'components/InfoBox';
import InlineEditor from 'components/InlineEditor';
import { relativeTimeRenderer } from 'components/Table';
import TagList from 'components/TagList';
import { ModelVersion } from 'types';
import { formatDatetime } from 'utils/date';

import css from './ModelVersionHeader.module.scss';

interface Props {
  modelVersion: ModelVersion;
}

const ModelVersionHeader: React.FC<Props> = ({ modelVersion }: Props) => {
  const infoRows: InfoRow[] = useMemo(() => {
    return [ {
      content: formatDatetime(modelVersion.creationTime, 'MMM DD, YYYY', false),
      label: 'Created',
    },
    { content: relativeTimeRenderer(new Date()), label: 'Updated' },
    { content: <InlineEditor placeholder="Add description..." value="" />, label: 'Description' },
    {
      content: <TagList
        ghost={false}
        tags={[]}
      />,
      label: 'Tags',
    } ] as InfoRow[];
  }, [ modelVersion ]);

  return (
    <header className={css.base}>
      <div className={css.breadcrumbs}>
        <Breadcrumb separator="">
          <Breadcrumb.Item href={`det/models/${modelVersion.model?.name}`}>
            <LeftOutlined style={{ marginRight: 10 }} />
          </Breadcrumb.Item>
          <Breadcrumb.Item href="det/models">Model Registry</Breadcrumb.Item>
          <Breadcrumb.Separator />
          <Breadcrumb.Item href={`det/models/${modelVersion.model?.name}`}>
            {modelVersion.model?.name}
          </Breadcrumb.Item>
          <Breadcrumb.Separator />
          <Breadcrumb.Item>Version {modelVersion.version}</Breadcrumb.Item>
        </Breadcrumb>
      </div>
      <div style={{
        marginLeft: 24,
        marginRight: 24,
      }}>
        <div className={css.mainRow}>
          <div className={css.title}>
            <div className={css.versionBox}>
              V{modelVersion.version}
            </div>
            <h1 className={css.versionName}>Version {modelVersion.version}</h1>
          </div>
          <div className={css.buttons}>
            <Button>Download Model</Button>
            <Button type="text"><Icon name="overflow-horizontal" size="tiny" /></Button>
          </div>
        </div>
        <InfoBox rows={infoRows} seperator={false} />
      </div>
    </header>
  );
};

export default ModelVersionHeader;
