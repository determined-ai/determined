import { ExclamationCircleOutlined } from '@ant-design/icons';
import { Modal as AntdModal } from 'antd';
import React, { Dispatch, ReactNode, SetStateAction, useCallback, useState } from 'react';

import Button from 'components/kit/Button';
import Link from 'components/Link';
import Icon from 'shared/components/Icon';

import css from './Modal.module.scss';

interface LinkParams {
  text: string;
  url: string;
}

export type ModalSize = 'small' | 'medium' | 'large';
export type Opener = Dispatch<SetStateAction<boolean>>;

export type ModalParams = {
  isOpen: boolean;
  setIsOpen: Opener;
};

export interface ModalSubmitParams {
  disabled?: boolean;
  text: string;
  handler: () => Promise<void>;
  onComplete?: () => Promise<void>;
}

interface ModalProps {
  cancelText: string;
  danger?: boolean;
  footerLink?: LinkParams;
  headerLink?: LinkParams;
  icon?: string;
  key?: string;
  size: ModalSize;
  submit: ModalSubmitParams;
  titleText: string;
  isOpen: boolean;
  setIsOpen: Opener;
  children: ReactNode;
}

export const Modal: React.FC<ModalProps> = ({
  isOpen,
  cancelText,
  danger,
  footerLink,
  headerLink,
  icon,
  key,
  size,
  submit,
  titleText,
  children: modalBody,
  setIsOpen,
}: ModalProps) => {
  const [isSubmitting, setIsSubmitting] = useState(false);

  const close = useCallback(() => setIsOpen(false), [setIsOpen]);

  const handleOk = useCallback(async () => {
    setIsSubmitting(true);
    await submit?.handler();
    setIsSubmitting(false);
    setIsOpen(false);
    await submit?.onComplete?.();
  }, [submit, setIsOpen]);

  return (
    <AntdModal
      cancelText={cancelText}
      footer={
        <div style={{ display: 'flex', justifyContent: 'space-between' }}>
          <div>
            {footerLink && (
              <Link path={footerLink.url} popout>
                {footerLink.text}
              </Link>
            )}
          </div>
          <div>
            <Button key="back" onClick={close}>
              {cancelText}
            </Button>
            <Button
              danger={danger}
              disabled={!!submit?.disabled}
              key="submit"
              loading={isSubmitting}
              type="primary"
              onClick={handleOk}>
              {submit.text}
            </Button>
          </div>
        </div>
      }
      key={key}
      maskClosable={true}
      open={isOpen}
      style={{
        minWidth: size === 'small' ? 358 : size === 'medium' ? 692 : 1025,
      }}
      title={
        <div style={{ display: 'flex' }}>
          {danger ? (
            <div className={css.dangerIcon} style={{ paddingRight: 16 }}>
              <ExclamationCircleOutlined />
            </div>
          ) : icon ? (
            <div style={{ paddingRight: 16 }}>
              <Icon name={icon} />
            </div>
          ) : null}
          <div style={{ paddingRight: 4 }}>{titleText}</div>
          {headerLink && (
            <Link path={headerLink.url} popout>
              {headerLink.text}
            </Link>
          )}
        </div>
      }
      onCancel={close}
      onOk={handleOk}>
      {modalBody}
    </AntdModal>
  );
};

export const useModal = <ModalProps extends { isOpen: boolean; setIsOpen: Opener }>(
  Comp: React.FC<ModalProps>,
): { Component: React.FC<Omit<ModalProps, 'isOpen' | 'setIsOpen'>>; open: () => void } => {
  const [modalIsOpen, setModalIsOpen] = useState(false);
  const handleOpen = React.useCallback(() => setModalIsOpen(true), []);

  const Component = React.useCallback(
    (props: Omit<ModalProps, 'isOpen' | 'setIsOpen'>) => {
      const p = { isOpen: modalIsOpen, setIsOpen: setModalIsOpen, ...props } as ModalProps;
      return <Comp {...p} />;
    },
    [Comp, modalIsOpen],
  );
  return { Component, open: handleOpen };
};
