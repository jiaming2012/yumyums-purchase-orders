Feature: Manager invites team member and new user completes onboarding

  Scenario: Manager invites a new team member who accepts and sees permitted apps
    # Device A: Manager configures permissions
    Given User A is logged in on Device A as "jamal@yumyums.kitchen"
    And User A navigates to the Users app
    And User A opens the Access tab
    And User A enables "Onboarding" for team members
    And User A enables "Operations" for team members

    # Device A: Manager creates the invite
    When User A clicks "Add Crew Member"
    And User A fills in the invite form:
      | field           | value            |
      | first name      | Jim              |
      | last name       | Brown            |
      | email           | jim@gmail.com    |
      | role            | Team Member      |
      | employee type   | 1099             |
      | starting salary | 20               |
    And User A submits the invite
    Then User A should see the invite link panel

    # Device B: New user accepts invite
    When User A copies the invite link
    And User B opens the invite link on Device B
    Then User B should see the welcome form

    When User B fills in the onboarding form:
      | field        | value      |
      | password     | test74774  |
      | toast pos    | 1234       |
      | cashapp id   | deviceB    |
      | phone number | 5559483728 |
    And User B submits the onboarding form
    Then User B should be redirected to the home screen

    # Device B: Verify permissions
    And User B should see exactly 2 app tiles
    And User B should see the "Onboarding" app
    And User B should see the "Operations" app
