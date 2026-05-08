@setup @pat
Feature: GitHub Personal Access Token の検証
  初回セットアップで GitHub PAT を入力し、有効性を確認できる

  Background:
    Given ユーザーがセットアップページを開いている

  Scenario: 有効な PAT を入力すると次のステップに進む
    When 有効な PAT を入力して "検証" ボタンを押す
    Then ステップ 2 (リポジトリ選択) に進む

  Scenario: 無効な PAT を入力するとエラーが表示される
    When 無効な PAT を入力して "検証" ボタンを押す
    Then エラーメッセージが表示される
    And ステップ 1 のままである
