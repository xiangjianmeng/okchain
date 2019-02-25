pragma solidity ^0.4.0;

contract bank {

    address deployer;
    uint64 total_token;
    mapping(address => uint64) balance_map;

    event NegtiveTokensNotAllow(address from, uint64 deposit_token);
    event BalanceNotEnough(address from, uint64 deposit_token);
    event Info(address from, string msg, uint64 tokens);

    constructor() public payable {
        deployer = msg.sender;
        total_token = 0;
    }

    function deposit_token(uint64 tokens) public returns (uint64) {
        balance_map[msg.sender] += tokens;
        total_token += tokens;

        emit Info(msg.sender, "success to deposit ", tokens);
        return balance_map[msg.sender];
    }

    // if return 0, no token withdrawed; if return == tokens, withdraw success;
    function withdraw_token(uint64 tokens) public returns (uint64) {
        var balance = balance_map[msg.sender];
        if (tokens > balance) {
            emit BalanceNotEnough(msg.sender, tokens);
            return 0;
        } else {
            balance_map[msg.sender] -= tokens;
            total_token -= tokens;
            emit Info(msg.sender, "success to withdraw ", tokens);
            return tokens;
        }
    }

    function get_total_token() public view returns (uint64) {
        return total_token;
    }

    function get_token_count() public view returns (uint64) {
        return balance_map[msg.sender];
    }

}