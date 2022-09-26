import { useCallback } from '@storybook/addons';
import { Button, Space } from 'antd';
import { ModalFuncProps } from 'antd/es/modal/Modal';
import React, { useMemo } from 'react';

import loremIpsum from 'shared/utils/loremIpsum';

import useModal from './useModal';
import css from './useModal.stories.module.scss';

export default {
  component: useModal,
  title: 'Shared/Modal',
};

const args: Partial<ModalFuncProps> = {
  cancelText: 'Cancel',
  closable: true,
  maskClosable: true,
  okText: 'Ok',
  title: 'Modal Title',
  width: undefined,
};

export const Default = (args: Partial<ModalFuncProps>): React.ReactNode => {
  const { contextHolder, modalOpen } = useModal();

  const modalProps: Partial<ModalFuncProps> = useMemo(() => {
    return {
      content: loremIpsum,
      icon: null,
      ...args,
    };
  }, [args]);

  return (
    <>
      <Button onClick={() => modalOpen(modalProps)}>Open Modal</Button>
      {contextHolder}
    </>
  );
};

export const SeparatedBody = (args: Partial<ModalFuncProps>): React.ReactNode => {
  const { contextHolder, modalOpen, modalClose } = useModal();

  const handleCloseModal = useCallback(() => {
    modalClose();
  }, [modalClose]);

  const modalProps: Partial<ModalFuncProps> = useMemo(() => {
    return {
      className: css.borderedModal,
      content: (
        <>
          <div className={css.base}>{loremIpsum}</div>
          <div className={css.footer}>
            <div className={css.spacer} />
            <Space>
              <Button onClick={handleCloseModal}>{args.cancelText}</Button>
              <Button type="primary" onClick={handleCloseModal}>
                {args.okText}
              </Button>
            </Space>
          </div>
        </>
      ),
      icon: null,
      ...args,
    };
  }, [args, handleCloseModal]);

  return (
    <>
      <Button onClick={() => modalOpen(modalProps)}>Open Modal</Button>
      {contextHolder}
    </>
  );
};

export const ExtraFooterButton = (args: Partial<ModalFuncProps>): React.ReactNode => {
  const { contextHolder, modalOpen, modalClose } = useModal();

  const handleCloseModal = useCallback(() => {
    modalClose();
  }, [modalClose]);

  const modalProps: Partial<ModalFuncProps> = useMemo(() => {
    return {
      className: css.footerModal,
      content: (
        <>
          <div className={css.base}>{loremIpsum}</div>
          <div className={css.footer}>
            <Button type="text">Extra Button</Button>
            <div className={css.spacer} />
            <Space>
              <Button onClick={handleCloseModal}>{args.cancelText}</Button>
              <Button type="primary" onClick={handleCloseModal}>
                {args.okText}
              </Button>
            </Space>
          </div>
        </>
      ),
      icon: null,
      ...args,
    };
  }, [args, handleCloseModal]);

  return (
    <>
      <Button onClick={() => modalOpen(modalProps)}>Open Modal</Button>
      {contextHolder}
    </>
  );
};

export const OneFooterButton = (args: Partial<ModalFuncProps>): React.ReactNode => {
  const { contextHolder, modalOpen } = useModal();

  const modalProps: Partial<ModalFuncProps> = useMemo(() => {
    return {
      content: loremIpsum,
      icon: null,
      ...args,
      cancelButtonProps: { style: { display: 'none' } },
    };
  }, [args]);

  return (
    <>
      <Button onClick={() => modalOpen(modalProps)}>Open Modal</Button>
      {contextHolder}
    </>
  );
};

export const NoFooter = (args: Partial<ModalFuncProps>): React.ReactNode => {
  const { contextHolder, modalOpen } = useModal();

  const modalProps: Partial<ModalFuncProps> = useMemo(() => {
    return {
      className: css.noFooterModal,
      content: loremIpsum,
      icon: null,
      ...args,
    };
  }, [args]);

  return (
    <>
      <Button onClick={() => modalOpen(modalProps)}>Open Modal</Button>
      {contextHolder}
    </>
  );
};

Default.args = args;
SeparatedBody.args = args;
ExtraFooterButton.args = args;
OneFooterButton.args = args;
NoFooter.args = args;
