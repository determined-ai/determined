import { Drawer } from 'antd';
import React from 'react';

import Button from 'components/kit/Button';
import css from 'components/kit/Drawer.module.scss';
import Icon from 'components/kit/Icon';

type DrawerPlacement = 'left' | 'right';

interface DrawerProps {
  children: React.ReactNode;
  maskClosable?: boolean;
  open: boolean;
  placement: DrawerPlacement;
  title: string;
  onClose: () => void;
}

const DrawerComponent: React.FC<DrawerProps> = ({
  children,
  maskClosable = true,
  open,
  placement,
  title,
  onClose,
}) => {
  return (
    <Drawer
      bodyStyle={{ padding: 0 }}
      closable={false}
      maskClosable={maskClosable}
      open={open}
      placement={placement}
      rootClassName={css.mobileWidth}
      width="700px"
      onClose={onClose}>
      <div className={css.header}>
        <div className={css.headerTitle}>{title}</div>
        <Button
          icon={<Icon name="close" size="small" title="Close drawer" />}
          type="text"
          onClick={onClose}
        />
      </div>
      <div className={css.body}>{children}</div>
    </Drawer>
  );
};

export default DrawerComponent;
