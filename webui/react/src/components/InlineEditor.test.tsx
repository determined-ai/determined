import {
    render,
    screen,
    waitForElementToBeRemoved,
  } from "@testing-library/react";
  import userEvent from "@testing-library/user-event";
  
  import React from "react";
  
  import InlineEditor from "./InlineEditor";
  
  const setup = (text: string = "", disabled = false) => {
    const onSave = jest.fn();
    const onCancel = jest.fn();
    const { container } = render(
      <InlineEditor
        value={text}
        onSave={onSave}
        onCancel={onCancel}
        disabled={disabled}
      />
    );
  
    const waitForSpinnerToDisappear = async () =>
      await waitForElementToBeRemoved(
        () => container.getElementsByClassName("ant-spin-spinning")[0]
      );
    return { onSave, onCancel, waitForSpinnerToDisappear };
  };
  
  describe("InlineEditor", () => {
    it("displays the value passed as prop", async () => {
      setup("before");
      expect(screen.getByDisplayValue("before")).toBeInTheDocument();
    });
  
    it("preserves input when focus leaves", async () => {
      const { waitForSpinnerToDisappear } = setup("before");
      userEvent.clear(screen.getByRole("textbox"));
      userEvent.type(screen.getByRole("textbox"), "after");
      userEvent.click(document.body);
      expect(screen.getByRole("textbox")).not.toHaveFocus();
      expect(screen.getByDisplayValue("after")).toBeInTheDocument();
  
      // wait for spinner to go away
      // to avoid "test not wrapped in act(...)"
      await waitForSpinnerToDisappear();
    });
  
    it("calls save with input on blur", async () => {
      const { onSave, waitForSpinnerToDisappear } = setup("before");
      userEvent.clear(screen.getByRole("textbox"));
      userEvent.type(screen.getByRole("textbox"), "after");
      userEvent.click(document.body);
  
      await waitForSpinnerToDisappear();
      expect(onSave).toHaveBeenCalledWith("after");
      expect(screen.getByRole("textbox")).not.toHaveFocus();
    });
  
    it("calls restores previous value when esc is pressed", async () => {
      setup("before");
      userEvent.clear(screen.getByRole("textbox"));
      userEvent.type(screen.getByRole("textbox"), "after{escape}");
      expect(screen.getByDisplayValue("before")).toBeInTheDocument();
    });
  
    it("doesnt allow user input when disabled", async () => {
      setup("before", true);
      userEvent.clear(screen.getByRole("textbox"));
      userEvent.type(screen.getByRole("textbox"), "after");
      expect(screen.getByDisplayValue("before")).toBeInTheDocument();
    });
  });
  