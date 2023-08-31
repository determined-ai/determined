import { Modal } from 'antd';
import { ModalProps } from 'antd/es/modal/Modal';
import React, { useCallback, useEffect, useState } from 'react';

import css from 'components/GalleryModal.module.scss';
import Button from 'components/kit/Button';
import Icon from 'components/kit/Icon';
import { keyEmitter, KeyEvent } from 'hooks/useKeyTracker';
import useResize from 'hooks/useResize';
import { isNumber } from 'utils/data';
import { isPercent, percentToFloat } from 'utils/number';

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
          <Button
            icon={<Icon name="arrow-left" showTooltip title="Previous" />}
            onClick={handlePrevious}
          />
        </div>
        <div className={css.next}>
          <Button
            icon={<Icon name="arrow-right" showTooltip title="Next" />}
            onClick={handleNext}
          />
        </div>
      </div>
    </Modal>
  );
};

export default GalleryModal;
