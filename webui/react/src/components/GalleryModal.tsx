import { Modal } from 'antd';
import { ModalProps } from 'antd/es/modal/Modal';
import React, { useCallback, useEffect, useState } from 'react';

import Button from 'components/kit/Button';
import Tooltip from 'components/kit/Tooltip';
import { keyEmitter, KeyEvent } from 'hooks/useKeyTracker';
import useResize from 'hooks/useResize';
import Icon from 'shared/components/Icon';
import { isNumber } from 'shared/utils/data';
import { isPercent, percentToFloat } from 'shared/utils/number';

import css from './GalleryModal.module.scss';

interface Props extends ModalProps {
  children: React.ReactNode;
  height?: number | string;
  onNext?: () => void;
  onPrevious?: () => void;
}

const GalleryModal: React.FC<Props> = ({
  height = '80%',
  onNext,
  onPrevious,
  children,
  ...props
}: Props) => {
  const resize = useResize();
  const [width, setWidth] = useState<number>();
  const [minHeight, setMinHeight] = useState<number>();

  const handlePrevious = useCallback(() => {
    if (onPrevious) onPrevious();
  }, [onPrevious]);

  const handleNext = useCallback(() => {
    if (onNext) onNext();
  }, [onNext]);

  useEffect(() => {
    setWidth(resize.width);

    if (isPercent(height)) {
      const newMinHeight = percentToFloat(height) * resize.height;
      setMinHeight(newMinHeight);
    } else if (isNumber(height) && height < resize.height) {
      setMinHeight(height);
    }
  }, [height, resize]);

  useEffect(() => {
    const keyUpListener = (e: KeyboardEvent) => {
      if (e.key === 'ArrowLeft') {
        if (onPrevious) onPrevious();
      } else if (e.key === 'ArrowRight') {
        if (onNext) onNext();
      }
    };

    keyEmitter.on(KeyEvent.KeyUp, keyUpListener);

    return () => {
      keyEmitter.off(KeyEvent.KeyUp, keyUpListener);
    };
  }, [onNext, onPrevious]);

  return (
    <Modal centered footer={null} open width={width} {...props}>
      <div className={css.base} style={{ minHeight }}>
        {children}
        <div className={css.prev}>
          <Tooltip placement="right" title="Previous">
            <Button onClick={handlePrevious}>
              <Icon name="arrow-left" />
            </Button>
          </Tooltip>
        </div>
        <div className={css.next}>
          <Tooltip placement="left" title="Next">
            <Button onClick={handleNext}>
              <Icon name="arrow-right" />
            </Button>
          </Tooltip>
        </div>
      </div>
    </Modal>
  );
};

export default GalleryModal;
