import { Modal } from 'antd';
import { ModalProps } from 'antd/es/modal/Modal';
import React, { PropsWithChildren, useCallback, useEffect, useState } from 'react';

import useResize from 'hooks/useResize';
import { isNumber } from 'utils/data';
import { isPercent, percentToFloat } from 'utils/number';

import css from './GalleryModal.module.scss';
import IconButton from './IconButton';

interface Props extends ModalProps {
  height?: number | string;
  onNext?: () => void;
  onPrevious?: () => void;
}

const GalleryModal: React.FC<PropsWithChildren<Props>> = ({
  height = '80%',
  onNext,
  onPrevious,
  children,
  ...props
}: PropsWithChildren<Props>) => {
  const resize = useResize();
  const [ width, setWidth ] = useState<number>();
  const [ minHeight, setMinHeight ] = useState<number>();

  const handlePrevious = useCallback(() => {
    if (onPrevious) onPrevious();
  }, [ onPrevious ]);

  const handleNext = useCallback(() => {
    if (onNext) onNext();
  }, [ onNext ]);

  useEffect(() => {
    setWidth(resize.width);

    if (isPercent(height)) {
      const newMinHeight = percentToFloat(height) * resize.height;
      setMinHeight(newMinHeight);
    } else if (isNumber(height) && height < resize.height) {
      setMinHeight(height);
    }
  }, [ height, resize ]);

  return (
    <Modal
      centered
      footer={null}
      visible
      width={width}
      {...props}>
      <div className={css.base} style={{ minHeight }}>
        {children}
        <IconButton
          className={css.prev}
          icon="arrow-left"
          iconSize="small"
          label="Previous"
          tooltipPlacement="right"
          onClick={handlePrevious}
        />
        <IconButton
          className={css.next}
          icon="arrow-right"
          iconSize="small"
          label="Next"
          tooltipPlacement="left"
          onClick={handleNext}
        />
      </div>
    </Modal>
  );
};

export default GalleryModal;
