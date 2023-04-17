// SPDX-License-Identifier: MIT
//
// Copyright (c) 2023 Berachain Foundation
//
// Permission is hereby granted, free of charge, to any person
// obtaining a copy of this software and associated documentation
// files (the "Software"), to deal in the Software without
// restriction, including without limitation the rights to use,
// copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the
// Software is furnished to do so, subject to the following
// conditions:
//
// The above copyright notice and this permission notice shall be
// included in all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
// EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES
// OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
// NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT
// HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY,
// WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
// FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR
// OTHER DEALINGS IN THE SOFTWARE.

pragma solidity ^0.8.4;

/**
 * @dev Interface of all supported Cosmos events emitted by the bank module
 */
interface IBankModule {
    ////////////////////////////////////////// EVENTS /////////////////////////////////////////////

    /**
     * @dev Emitted by the bank module when `amount` tokens are sent to `recipient`
     */
    event Transfer(address indexed recipient, uint256 amount);

    /**
     * @dev Emitted by the bank module when `sender` sends some amount of tokens
     */
    event Message(address indexed sender);

    /**
     * @dev Emitted by the bank module when `amount` tokens are spent by `spender`
     */
    event CoinSpent(address indexed spender, uint256 amount);

    /**
     * @dev Emitted by the bank module when `amount` tokens are received by `receiver`
     */
    event CoinReceived(address indexed receiver, uint256 amount);

    /**
     * @dev Emitted by the bank module when `amount` tokens are minted by `minter`
     *
     * Note: "Coinbase" refers to the Cosmos event: EventTypeCoinMint. `minter` is a module
     * address.
     */
    event Coinbase(address indexed minter, uint256 amount);

    /**
     * @dev Emitted by the bank module when `amount` tokens are burned by `burner`
     *
     * Note: `burner` is a module address
     */
    event Burn(address indexed burner, uint256 amount);

    /////////////////////////////////////// READ METHODS //////////////////////////////////////////

    /**
     * @dev Returns the `amount` of account balance by address for a given denomination.
     */
    function getBalance(address accountAddress, string calldata denom) external view returns (uint256);

    /**
     * @dev Returns account balance by address for all denominations.
     */
    function getAllBalances(address accountAddress) external view returns (Coin[] memory);

    /**
     * @dev Returns the `amount` of account balance by address for a given denomination.
     */
    function getSpendableBalance(address accountAddress, string calldata denom) external view returns (uint256);

    /**
     * @dev Returns account balance by address for all denominations.
     */
    function getAllSpendableBalances(address accountAddress) external view returns (Coin[] memory);

    /**
     * @dev Returns the total supply of a single coin.
     */
    function getSupply(string calldata denom) external view returns (uint256);

    /**
     * @dev Returns the total supply of a all coins.
     */
    function getAllSupply() external view returns (Coin[] memory);

    /**
     * @dev Returns the denomination's metadata.
     */
    function getDenomMetadata(string calldata denom) external view returns (DenomMetadata memory);

    /**
     * @dev Returns if the denom is enabled to send
     */
    function getSendEnabled(string calldata denom) external view returns (bool);

    ////////////////////////////////////// WRITE METHODS //////////////////////////////////////////

    /**
     * @dev Send coins from one address to another.
     */
    function send(address fromAddress, address toAddress, Coin[] calldata amount) external payable returns (bool);

    //////////////////////////////////////////// UTILS ////////////////////////////////////////////

    /**
     * @dev Represents a cosmos coin.
     * Note: this struct is generated as go struct that is then used in the precompile.
     */
    struct Coin {
        uint256 amount;
        string denom;
    }

    /**
     * @dev Represents a denom unit.
     * Note: this struct is generated in generated/i_bank_module.abigen.go
     */
    struct DenomUnit {
        string denom;
        string[] aliases;
        uint32 exponent;
    }

    /**
     * @dev Represents a denom metadata.
     * Note: this struct is generated in generated/i_bank_module.abigen.go
     */
    struct DenomMetadata {
        string description;
        DenomUnit[] denomUnits;
        string base;
        string display;
        string name;
        string symbol;
    }
}
