import { Drawer } from 'antd';
import React from 'react';

import Button from 'components/kit/Button';
import Icon from 'components/kit/Icon';

import css from './Drawer.module.scss';
import { DarkLight } from './internal/types';
import useUI, { UIProvider } from './Theme';

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
  const {
    ui: { mode, theme },
  } = useUI();

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
      <UIProvider darkMode={mode === DarkLight.Dark} theme={theme}>
        <div className={css.header}>
          <div className={css.headerTitle}>{title}</div>
          <Button
            icon={<Icon name="close" size="small" title="Close drawer" />}
            type="text"
            onClick={onClose}
          />
        </div>
        <div className={css.body}>{children}</div>
      </UIProvider>
    </Drawer>
  );
};

export default DrawerComponent;
