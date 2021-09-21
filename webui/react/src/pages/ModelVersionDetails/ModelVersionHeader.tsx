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
    <header style={{
      backgroundColor: 'var(--theme-colors-monochrome-17)',
      margin: 0,
      padding: 12,
      width: '100%',
    }}>
      <div style={{
        borderBottom: '1px solid var(--theme-colors-monochrome-12)',
        paddingBottom: 8,
      }}>
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
        <div style={{
          alignItems: 'center',
          display: 'flex',
          justifyContent: 'space-between',
        }}>
          <div>
            <img />
            <h1>Version {modelVersion.version}</h1>
          </div>
          <div style={{ display: 'flex', gap: 4 }}>
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
