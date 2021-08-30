import { DownOutlined } from '@ant-design/icons';
import { Card } from 'antd';
import React, { PropsWithChildren, useMemo, useState } from 'react';

interface CollapsableCardProps {
  title?: React.ReactNode;
}

const CollapsableCard: React.FC<CollapsableCardProps> =
(props: PropsWithChildren<CollapsableCardProps>) => {
  const [ collapsed, setCollapsed ] = useState(false);

  const title = useMemo(() => {
    return <div style={{ alignItems: 'center', display: 'flex', gap: 4 }}>
      <DownOutlined
        style={{ transform: collapsed ? 'none' : 'rotate(180deg)' }}
        type="text"
        onClick={() => setCollapsed(prev => !prev)} />
      {props.title}
    </div>;
  }, [ collapsed, props.title ]);

  return (
    <Card bodyStyle={{ padding: 0 }} title={title}>
      {collapsed ? null : <div style={{ padding: 24 }}>{props.children}</div>}
    </Card>
  );
};

export default CollapsableCard;
