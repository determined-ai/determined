
import { Step, Table, BeforeSuite, AfterSuite } from "gauge-ts";
import { strictEqual } from 'assert';
import { checkBox, click, closeBrowser, evaluate, goto, into, link, openBrowser, press, text, textBox, toLeftOf, write } from 'taiko';
import assert = require("assert");

export default class StepImplementation {
    @BeforeSuite()
    public async beforeSuite() {
        await openBrowser({ headless: false });
    }

    @AfterSuite()
    public async afterSuite() {
        await closeBrowser();
    };

    @Step("Open todo application")
    public async openTodo() {
        await goto("todo.taiko.dev");
    }

    @Step("Add task <item>")
    public async addTask(item: string) {
        await write(item, into(textBox({
            class: "new-todo"
        })));
        await press('Enter');
    }

    @Step("Must display <message>")
    public async checkDisplay(message: string) {
        assert.ok(await text(message).exists(0, 0));
    }

    @Step("Add tasks <table>")
    public async addTasks(table: Table) {
        for (var row of table.getTableRows()) {
            await write(row.getCell("description"));
            await press('Enter');
        }
    }

    @Step("Complete tasks <table>")
    public async completeTasks(table: Table) {
        for (var row of table.getTableRows()) {
            await click(checkBox(toLeftOf(row.getCell("description"))));
        }
    }

    @Step("View <type> tasks")
    public async viewTasks(type: string) {
        await click(link(type));
    }

    @Step("Must have <table>")
    public async mustHave(table: Table) {
        for (var row of table.getTableRows()) {
            assert.ok(await text(row.getCell("description")).exists());
        }
    }

    @Step("Must not have <table>")
    public async mustNotHave(table: Table) {
        for (var row of table.getTableRows()) {
            assert.ok(!await text(row.getCell("description")).exists(0, 0));
        }
    }

    @Step("Clear all tasks")
    public async clearAllTasks() {
        // @ts-ignore
        await evaluate(() => localStorage.clear());
    }

}
