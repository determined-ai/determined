import Button from 'hew/Button';
import Icon from 'hew/Icon';
import { Modal } from 'hew/Modal';
import React, { useCallback, useEffect } from 'react';

import { UPlotScatterProps } from 'components/UPlot/types';
import { keyEmitter, KeyEvent } from 'hooks/useKeyTracker';
import { Scale } from 'types';

import css from './GalleryModal.module.scss';
import UPlotScatter from './UPlot/UPlotScatter';

interface Props {
  onNext?: () => void;
  onPrevious?: () => void;
  onCancel: () => void;
  chartProps: Record<string, UPlotScatterProps> | undefined;
  activeHParam: string | undefined;
  selectedScale: Scale;
}

const GalleryModalComponent: React.FC<Props> = ({
  onNext,
  onPrevious,
  onCancel,
  chartProps,
  activeHParam,
  selectedScale,
}: Props) => {
  const handlePrevious = useCallback(() => {
    if (onPrevious) onPrevious();
  }, [onPrevious]);

  const handleNext = useCallback(() => {
    if (onNext) onNext();
  }, [onNext]);

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
    <Modal
      size="large"
      submit={{
        handleError: () => {},
        handler: onCancel,
        text: 'Close',
      }}
      title=""
      onClose={onCancel}>
      <div className={css.base}>
        {chartProps && activeHParam && (
          <UPlotScatter
            colorScaleDistribution={selectedScale}
            data={chartProps[activeHParam].data}
            options={{
              ...chartProps[activeHParam].options,
              cursor: { drag: undefined },
              height: 400,
            }}
            tooltipLabels={chartProps[activeHParam].tooltipLabels}
          />
        )}
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

export default GalleryModalComponent;
