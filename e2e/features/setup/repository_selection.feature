@setup @repos
Feature: 計測対象リポジトリの選択
  PAT 検証後、取得したリポジトリ一覧から計測対象を選ぶ

  Background:
    Given ユーザーが PAT 検証を完了している
    And リポジトリ選択ステップ (ステップ 2) を表示している

  Scenario: リポジトリを検索して絞り込める
    When 検索ボックスに "fourkeys" と入力する
    Then 一覧には "fourkeys" を含むリポジトリだけが表示される

  Scenario: 1 つ以上選択するとグループ作成ステップに進める
    When リポジトリを 1 つチェックする
    And "次へ" ボタンを押す
    Then ステップ 3 (グループ作成) に進む
